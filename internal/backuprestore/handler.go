/*
Copyright 2018 The aerospike-operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backuprestore

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	batch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	batchlistersv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
	aerospikelisters "github.com/travelaudience/aerospike-operator/internal/client/listers/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/logfields"
	"github.com/travelaudience/aerospike-operator/internal/meta"
	"github.com/travelaudience/aerospike-operator/internal/utils/events"
)

type AerospikeBackupRestoreHandler struct {
	kubeclientset           kubernetes.Interface
	aerospikeclientset      aerospikeclientset.Interface
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister
	jobsLister              batchlistersv1.JobLister
	recorder                record.EventRecorder
}

func New(kubeclientset kubernetes.Interface,
	aerospikeclientset aerospikeclientset.Interface,
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister,
	jobsLister batchlistersv1.JobLister,
	recorder record.EventRecorder) *AerospikeBackupRestoreHandler {
	return &AerospikeBackupRestoreHandler{
		kubeclientset:           kubeclientset,
		aerospikeclientset:      aerospikeclientset,
		aerospikeClustersLister: aerospikeClustersLister,
		jobsLister:              jobsLister,
		recorder:                recorder,
	}
}

// Handle manages the lifecycle of the obj resource.
func (h *AerospikeBackupRestoreHandler) Handle(obj aerospikev1alpha2.BackupRestoreObject) error {
	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Debug("checking whether action is needed")

	// check if the current resource is already marked as failed/finished, in
	// which case we return immediately
	if h.isFailedOrFinished(obj) {
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debug("no action is needed")
		if obj.SyncStatusWithSpec() {
			if err := h.updateStatus(obj); err != nil {
				return err
			}
		}
		return h.clearSecrets(obj)
	}

	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Infof("processing %s", obj.GetOperationType())

	// get backupstoragespec from the "parent" aerospikecluster resource in case
	// this field is not specified in the current resource
	if obj.GetStorage() == nil {
		aerospikeCluster, err := h.aerospikeClustersLister.AerospikeClusters(obj.GetNamespace()).Get(obj.GetTarget().Cluster)
		if err != nil {
			return err
		}
		obj.SetStorage(&aerospikeCluster.Spec.BackupSpec.Storage)
	}

	// check whether the associated job exists, and create it if it doesn't
	job, err := h.jobsLister.Jobs(obj.GetObjectMeta().Namespace).Get(h.getJobName(obj))
	if err != nil {
		if errors.IsNotFound(err) {
			// get the secret containing the credentials to access cloud storage
			secret, err := h.getSecret(obj)
			if err != nil {
				return err
			}
			// the job doesn't exist yet, so create it
			if err := h.launchJob(obj, secret); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		// at this point there is already an associated job, so we must check its
		// status and report accordingly
		h.maybeSetConditions(obj, job)
	}
	// sync .status with .spec
	obj.SyncStatusWithSpec()
	return h.updateStatus(obj)
}

// launchJob performs a number of checks and launches the job associated with
// obj.
func (h *AerospikeBackupRestoreHandler) launchJob(obj aerospikev1alpha2.BackupRestoreObject, secret *corev1.Secret) error {
	// create the backup/restore job
	job, err := h.createJob(obj, secret)
	if err != nil {
		return err
	}

	// log that the job has been created
	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Debugf("%s job created as %s", obj.GetOperationType(), meta.Key(job))
	// record an event indicating the current status
	h.recorder.Eventf(obj.(runtime.Object),
		corev1.EventTypeNormal, events.ReasonJobCreated,
		"%s job created as %s", obj.GetOperationType(), meta.Key(job))
	// append a condition to the resource indicating the current status
	condition := apiextensions.CustomResourceDefinitionCondition{
		LastTransitionTime: metav1.NewTime(time.Now()),
		Type:               obj.GetStartedConditionType(),
		Status:             apiextensions.ConditionTrue,
		Message:            fmt.Sprintf("%s job created as %s", obj.GetOperationType(), meta.Key(job)),
	}
	obj.SetConditions(append(obj.GetConditions(), condition))
	return nil
}

// maybeSetConditions checks the status of the job associated with obj and updates the
// resource's conditions.
func (h *AerospikeBackupRestoreHandler) maybeSetConditions(obj aerospikev1alpha2.BackupRestoreObject, job *batch.Job) {
	var jobCondition batch.JobConditionType

	// look for the complete or failed condition in the associated job
	for _, c := range job.Status.Conditions {
		if c.Type == batch.JobComplete && c.Status == corev1.ConditionTrue {
			jobCondition = batch.JobComplete
			break
		}
		if c.Type == batch.JobFailed && c.Status == corev1.ConditionTrue {
			jobCondition = batch.JobFailed
			break
		}
	}

	// update the resource's status based on the job condition
	switch jobCondition {
	case batch.JobComplete:
		// log that the job was successful
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debugf("%s job has finished", obj.GetOperationType())
		// record an event indicating success
		h.recorder.Eventf(obj.(runtime.Object), corev1.EventTypeNormal, events.ReasonJobFinished,
			"%s job has finished", obj.GetOperationType())
		// append a jobCondition to the resource's status indicating success
		obj.SetConditions(append(obj.GetConditions(), apiextensions.CustomResourceDefinitionCondition{
			LastTransitionTime: metav1.NewTime(time.Now()),
			Type:               obj.GetFinishedConditionType(),
			Status:             apiextensions.ConditionTrue,
			Message:            fmt.Sprintf("%s job has finished", obj.GetOperationType()),
		}))
	case batch.JobFailed:
		// log that the job failed
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debugf("%s job failed %d times", obj.GetOperationType(), job.Status.Failed)
		// record an event indicating failure
		h.recorder.Eventf(obj.(runtime.Object), corev1.EventTypeWarning, events.ReasonJobFailed,
			"%s job failed %d times", obj.GetOperationType(), job.Status.Failed)
		// append a jobCondition to the resource's status indicating failure
		obj.SetConditions(append(obj.GetConditions(), apiextensions.CustomResourceDefinitionCondition{
			LastTransitionTime: metav1.NewTime(time.Now()),
			Type:               obj.GetFailedConditionType(),
			Status:             apiextensions.ConditionTrue,
			Message:            fmt.Sprintf("%s job failed %d times", obj.GetOperationType(), job.Status.Failed),
		}))
	}
}
