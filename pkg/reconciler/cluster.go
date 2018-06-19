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

package reconciler

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	storagelistersv1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/record"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

type AerospikeClusterReconciler struct {
	kubeclientset      kubernetes.Interface
	aerospikeclientset aerospikeclientset.Interface
	podsLister         listersv1.PodLister
	configMapsLister   listersv1.ConfigMapLister
	servicesLister     listersv1.ServiceLister
	pvcsLister         listersv1.PersistentVolumeClaimLister
	scsLister          storagelistersv1.StorageClassLister
	recorder           record.EventRecorder
}

func New(kubeclientset kubernetes.Interface,
	aerospikeclientset aerospikeclientset.Interface,
	podsLister listersv1.PodLister,
	configMapsLister listersv1.ConfigMapLister,
	servicesLister listersv1.ServiceLister,
	pvcsLister listersv1.PersistentVolumeClaimLister,
	scsLister storagelistersv1.StorageClassLister,
	recorder record.EventRecorder) *AerospikeClusterReconciler {
	return &AerospikeClusterReconciler{
		kubeclientset:      kubeclientset,
		aerospikeclientset: aerospikeclientset,
		podsLister:         podsLister,
		configMapsLister:   configMapsLister,
		servicesLister:     servicesLister,
		pvcsLister:         pvcsLister,
		scsLister:          scsLister,
		recorder:           recorder,
	}
}

// MaybeReconcile checks if reconciliation is needed.
func (r *AerospikeClusterReconciler) MaybeReconcile(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debug("checking whether reconciliation is needed")

	// check if a previous upgrade operation has failed, in which case we return
	if v, ok := aerospikeCluster.ObjectMeta.Annotations[UpgradeStatusAnnotationKey]; ok {
		if v == UpgradeStatusFailedAnnotationValue {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			}).Warn("a previous version upgrade has failed. aborting")
			return nil
		}
	}

	// check if the current reconcile operation is an upgrade, and if it is set
	// the appropriate annotations (for internal use) and conditions
	upgrade := aerospikeCluster.Status.Version != "" && aerospikeCluster.Spec.Version != aerospikeCluster.Status.Version
	if upgrade {
		var err error
		aerospikeCluster, err = r.signalUpgradeStarted(aerospikeCluster)
		if err != nil {
			return err
		}
	}

	// validate fields that cannot be validated statically
	valid, err := r.validate(aerospikeCluster)
	if err != nil {
		return err
	}
	// if the resource is not valid, no reconciliation is performed and we may quit
	if !valid {
		return nil
	}
	// create the service for the cluster
	if err := r.ensureService(aerospikeCluster); err != nil {
		return err
	}
	// create/get the configmap
	configMap, err := r.ensureConfigMap(aerospikeCluster)
	if err != nil {
		return err
	}
	// create the network policy
	if err := r.ensureNetworkPolicy(aerospikeCluster); err != nil {
		return err
	}

	// make sure that pods are up-to-date with the spec
	if err := r.ensurePods(aerospikeCluster, configMap, upgrade); err != nil {
		// if a pod upgrade failed, signal with the appropriate annotations
		// and conditions
		if err == errors.PodUpgradeFailed {
			if _, err := r.signalUpgradeFailed(aerospikeCluster); err != nil {
				log.Errorf("failed to signal failed upgrade: %v", err)
			}
		}
		// return the original error
		return err
	}

	// update the status field of aerospikeCluster
	if err := r.updateStatus(aerospikeCluster); err != nil {
		return err
	}

	// set the appropriate annotations and conditions if performing an upgrade
	if upgrade {
		if _, err := r.signalUpgradeFinished(aerospikeCluster); err != nil {
			return err
		}
	}

	return nil
}
