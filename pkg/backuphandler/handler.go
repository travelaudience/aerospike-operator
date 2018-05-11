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

package backuphandler

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	batchlistersv1 "k8s.io/client-go/listers/batch/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	aerospikeErrors "github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
)

type AerospikeBackupsHandler struct {
	kubeclientset           kubernetes.Interface
	aerospikeclientset      aerospikeclientset.Interface
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister
	jobsLister              batchlistersv1.JobLister
	secretsLister           corelistersv1.SecretLister
	recorder                record.EventRecorder
}

func New(kubeclientset kubernetes.Interface,
	aerospikeclientset aerospikeclientset.Interface,
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister,
	jobsLister batchlistersv1.JobLister,
	secretsLister corelistersv1.SecretLister,
	recorder record.EventRecorder) *AerospikeBackupsHandler {
	return &AerospikeBackupsHandler{
		kubeclientset:           kubeclientset,
		aerospikeclientset:      aerospikeclientset,
		aerospikeClustersLister: aerospikeClustersLister,
		jobsLister:              jobsLister,
		secretsLister:           secretsLister,
		recorder:                recorder,
	}
}

type BackupRestoreObject struct {
	Action     actionType
	Type       string
	Obj        interface{}
	Name       string
	Namespace  string
	UID        types.UID
	ObjectMeta *metav1.ObjectMeta
	Storage    *aerospikev1alpha1.BackupStorageSpec
	Target     *aerospikev1alpha1.TargetNamespace
}

func (h *AerospikeBackupsHandler) Handle(objInt interface{}) error {
	switch obj := objInt.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		return h.handle(&BackupRestoreObject{
			Action:     backupAction,
			Type:       fmt.Sprintf("%s%s", logfields.AerospikeNamespace, backupAction),
			Obj:        obj,
			Name:       obj.Name,
			Namespace:  obj.Namespace,
			UID:        obj.UID,
			ObjectMeta: &obj.ObjectMeta,
			Storage:    &obj.Spec.Storage,
			Target:     &obj.Spec.Target,
		})

	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		return h.handle(&BackupRestoreObject{
			Action:     restoreAction,
			Type:       fmt.Sprintf("%s%s", logfields.AerospikeNamespace, restoreAction),
			Obj:        obj,
			Name:       obj.Name,
			Namespace:  obj.Namespace,
			UID:        obj.UID,
			ObjectMeta: &obj.ObjectMeta,
			Storage:    &obj.Spec.Storage,
			Target:     &obj.Spec.Target,
		})
	}
	return fmt.Errorf("invalid type")
}

func (h *AerospikeBackupsHandler) handle(obj *BackupRestoreObject) error {
	log.WithFields(log.Fields{
		obj.Type: meta.Key(obj.Obj),
	}).Debug("checking whether action is needed")

	// Check if job is already completed
	if h.getConditionStatus(obj, aerospikev1alpha1.ConditionCompleted) == apiextensions.ConditionTrue {
		log.WithFields(log.Fields{
			obj.Type: meta.Key(obj.Obj),
		}).Debugf("%s job is already completed", obj.Action)
		return nil
	}

	// Check the job status
	if status, err := h.getJobStatus(obj); err != nil {
		if err != aerospikeErrors.JobDoesNotExist {
			return err
		}
	} else {
		if status.Succeeded > 0 {
			if err := h.setConditions(obj, map[apiextensions.CustomResourceDefinitionConditionType]apiextensions.ConditionStatus{
				aerospikev1alpha1.ConditionCompleted: apiextensions.ConditionTrue,
			}); err != nil {
				return err
			}
			log.WithFields(log.Fields{
				obj.Type: meta.Key(obj.Obj),
			}).Debugf("%s job completed with success", obj.Action)
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeNormal, events.ReasonJobCompleted,
				"%s job completed with success", obj.Action)
			return nil
		}
		if status.Active > 0 {
			log.WithFields(log.Fields{
				obj.Type: meta.Key(obj.Obj),
			}).Debugf("%s job is running", obj.Action)
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeNormal, events.ReasonJobRunning,
				"%s job is running", obj.Action)
		}
		if status.Failed > 0 {
			log.WithFields(log.Fields{
				obj.Type: meta.Key(obj.Obj),
			}).Debugf("%s job failed", obj.Action)
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeWarning, events.ReasonJobFailed,
				"%s job failed %d times", obj.Action, status.Failed)
		}
		return nil
	}

	if err := h.ensureClusterExists(obj); err != nil {
		if errors.IsNotFound(err) {
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidTarget,
				"cluster %s does not exist",
				obj.Target.Cluster,
			)
		}
		if err == aerospikeErrors.NamespaceDoesNotExist {
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidTarget,
				"cluster %s does not contain a namespace named %s",
				obj.Target.Cluster,
				obj.Target.Namespace,
			)
		}
		return err
	}

	if err := h.ensureSecretExists(obj); err != nil {
		if errors.IsNotFound(err) {
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidSecret,
				"specified secret does not exist",
			)
		}
		if err == aerospikeErrors.InvalidSecretFileName {
			h.recorder.Eventf(obj.Obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidSecret,
				"secret does not contain expected file (Expected \"%s\")", secretFileName,
			)
		}
		return err
	}

	if err := h.createJob(obj); err != nil {
		return err
	}

	if err := h.setConditions(obj, map[apiextensions.CustomResourceDefinitionConditionType]apiextensions.ConditionStatus{
		aerospikev1alpha1.ConditionCompleted: apiextensions.ConditionFalse,
		aerospikev1alpha1.ConditionCreated:   apiextensions.ConditionTrue,
		aerospikev1alpha1.ConditionExpired:   apiextensions.ConditionFalse,
	}); err != nil {
		return err
	}
	return nil
}
