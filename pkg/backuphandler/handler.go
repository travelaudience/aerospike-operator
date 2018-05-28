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
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	batchlistersv1 "k8s.io/client-go/listers/batch/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/travelaudience/aerospike-operator/pkg/logfields"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	aerospikeerrors "github.com/travelaudience/aerospike-operator/pkg/errors"
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

func (h *AerospikeBackupsHandler) Handle(obj aerospikev1alpha1.BackupRestoreObject) error {
	log.WithFields(log.Fields{
		logfields.Kind: obj.GetKind(),
		logfields.Key:  meta.Key(obj),
	}).Debug("checking whether action is needed")

	// check if job is already completed
	if h.getConditionStatus(obj, aerospikev1alpha1.ConditionCompleted) == apiextensions.ConditionTrue {
		log.WithFields(log.Fields{
			logfields.Kind: obj.GetKind(),
			logfields.Key:  meta.Key(obj),
		}).Debugf("%s job is already completed", obj.GetAction())
		return nil
	}

	// check the job status
	if status, err := h.getJobStatus(obj); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	} else {
		if status.Succeeded > 0 {
			if err := h.appendCondition(obj, aerospikev1alpha1.ConditionCompleted, apiextensions.ConditionTrue); err != nil {
				return err
			}
			log.WithFields(log.Fields{
				logfields.Kind: obj.GetKind(),
				logfields.Key:  meta.Key(obj),
			}).Debugf("%s job completed", obj.GetAction())
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeNormal, events.ReasonJobCompleted,
				"%s job completed", obj.GetAction())
			return nil
		}
		if status.Active > 0 {
			log.WithFields(log.Fields{
				logfields.Kind: obj.GetKind(),
				logfields.Key:  meta.Key(obj),
			}).Debugf("%s job is running", obj.GetAction())
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeNormal, events.ReasonJobRunning,
				"%s job is running", obj.GetAction())
		}
		if status.Failed > 0 {
			log.WithFields(log.Fields{
				logfields.Kind: obj.GetKind(),
				logfields.Key:  meta.Key(obj),
			}).Debugf("%s job failed %d times", obj.GetAction(), status.Failed)
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonJobFailed,
				"%s job failed %d times", obj.GetAction(), status.Failed)
		}
		return nil
	}

	if err := h.checkNamespaceExists(obj); err != nil {
		if errors.IsNotFound(err) {
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidTarget,
				"cluster %s does not exist",
				obj.GetTarget().Cluster,
			)
		}
		if err == aerospikeerrors.NamespaceDoesNotExist {
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidTarget,
				"cluster %s does not contain a namespace named %s",
				obj.GetTarget().Cluster,
				obj.GetTarget().Namespace,
			)
		}
		return err
	}

	if err := h.checkSecretExists(obj); err != nil {
		if errors.IsNotFound(err) {
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidSecret,
				"secret does not exist",
			)
		}
		if err == aerospikeerrors.InvalidSecretFileName {
			h.recorder.Eventf(obj.(runtime.Object), v1.EventTypeWarning, events.ReasonInvalidSecret,
				"secret does not contain expected field %s", secretFileName,
			)
		}
		return err
	}

	if err := h.createJob(obj); err != nil {
		return err
	}
	if err := h.appendCondition(obj, aerospikev1alpha1.ConditionCreated, apiextensions.ConditionTrue); err != nil {
		return err
	}
	return nil
}
