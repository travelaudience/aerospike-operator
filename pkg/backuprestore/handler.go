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

	log "github.com/sirupsen/logrus"
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	batchlistersv1 "k8s.io/client-go/listers/batch/v1"
	"k8s.io/client-go/tools/record"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
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
func (h *AerospikeBackupRestoreHandler) Handle(obj aerospikev1alpha1.BackupRestoreObject) error {
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
		return nil
	}

	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Infof("processing %s", obj.GetAction())

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
	j, err := h.jobsLister.Jobs(obj.GetObjectMeta().Namespace).Get(h.getJobName(obj))
	if err != nil {
		if errors.IsNotFound(err) {
			// the job doesn't exist yet, so create it
			return h.launchJob(obj)
		}
		return err
	}

	// at this point there is already an associated job, so we must check its
	// status and report accordingly
	return h.updateStatus(obj, j)
}

// launchJob performs a number of checks and launches the job associated with
// obj.
func (h *AerospikeBackupRestoreHandler) launchJob(obj aerospikev1alpha1.BackupRestoreObject) error {
	// create the backup/restore job
	job, err := h.createJob(obj)
	if err != nil {
		return err
	}

	// log that the job has been created
	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Debugf("%s job created as %s", obj.GetAction(), meta.Key(job))
	// record an event indicating the current status
	h.recorder.Eventf(obj.(runtime.Object),
		v1.EventTypeNormal, events.ReasonJobCreated,
		"%s job created as %s", obj.GetAction(), meta.Key(job))
	// append a condition to the resource indicating the current status
	condition := apiextensions.CustomResourceDefinitionCondition{
		Type:    obj.GetStartedConditionType(),
		Status:  apiextensions.ConditionTrue,
		Message: fmt.Sprintf("%s job created as %s", obj.GetAction(), meta.Key(job)),
	}
	return h.appendCondition(obj, condition)
}

// updateStatus checks the status of the job associated with obj and updates the
// resource's status.
func (h *AerospikeBackupRestoreHandler) updateStatus(obj aerospikev1alpha1.BackupRestoreObject, job *batch.Job) error {
	var condition batch.JobConditionType

	// look for the complete of failed condition in the associated job
	for _, c := range job.Status.Conditions {
		if c.Type == batch.JobComplete && c.Status == v1.ConditionTrue {
			condition = batch.JobComplete
			break
		}
		if c.Type == batch.JobFailed && c.Status == v1.ConditionTrue {
			condition = batch.JobFailed
			break
		}
	}

	// update the resource's status based on the job condition
	switch condition {
	case batch.JobComplete:
		// log that the job was successful
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debugf("%s job has finished", obj.GetAction())
		// record an event indicating success
		h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeNormal, events.ReasonJobFinished,
			"%s job has finished", obj.GetAction())
		// append a condition to the resource's status indicating success
		condition := apiextensions.CustomResourceDefinitionCondition{
			Type:    obj.GetFinishedConditionType(),
			Status:  apiextensions.ConditionTrue,
			Message: fmt.Sprintf("%s job has finished", obj.GetAction()),
		}
		return h.appendCondition(obj, condition)
	case batch.JobFailed:
		// log that the job failed
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debugf("%s job failed %d times", obj.GetAction(), job.Status.Failed)
		// record an event indicating failure
		h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonJobFailed,
			"%s job failed %d times", obj.GetAction(), job.Status.Failed)
		// append a condition to the resource's status indicating failure
		condition := apiextensions.CustomResourceDefinitionCondition{
			Type:    obj.GetFailedConditionType(),
			Status:  apiextensions.ConditionTrue,
			Message: fmt.Sprintf("%s job failed %d times", obj.GetAction(), job.Status.Failed),
		}
		return h.appendCondition(obj, condition)
	default:
		return nil
	}
}
