/*
Copyright 2018 The aerospike-controller Authors.

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
	"k8s.io/client-go/tools/record"

	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

type AerospikeClusterReconciler struct {
	kubeclientset      kubernetes.Interface
	aerospikeclientset aerospikeclientset.Interface
	podsLister         listersv1.PodLister
	configMapsLister   listersv1.ConfigMapLister
	servicesLister     listersv1.ServiceLister
	recorder           record.EventRecorder
}

func New(kubeclientset kubernetes.Interface,
	aerospikeclientset aerospikeclientset.Interface,
	podsLister listersv1.PodLister,
	configMapsLister listersv1.ConfigMapLister,
	servicesLister listersv1.ServiceLister,
	recorder record.EventRecorder) *AerospikeClusterReconciler {
	return &AerospikeClusterReconciler{
		kubeclientset:      kubeclientset,
		aerospikeclientset: aerospikeclientset,
		podsLister:         podsLister,
		configMapsLister:   configMapsLister,
		servicesLister:     servicesLister,
		recorder:           recorder,
	}
}

// MaybeReconcile checks if reconciliation is needed.
func (r *AerospikeClusterReconciler) MaybeReconcile(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debug("checking whether reconciliation is needed")

	// validate fields that cannot be validated statically
	valid, err := r.validate(aerospikeCluster)
	if err != nil {
		return err
	}
	// if the resource is not valid, no reconciliation is performed and we may quit
	if !valid {
		return nil
	}

	// create the configmap
	if err := r.ensureConfigMap(aerospikeCluster); err != nil {
		return err
	}
	// create the client service for the cluster
	if err := r.ensureClientService(aerospikeCluster); err != nil {
		return err
	}
	// create the headless service for the cluster
	if err := r.ensureHeadlessService(aerospikeCluster); err != nil {
		return err
	}
	// make sure that current size meets desired size
	if err := r.ensureSize(aerospikeCluster); err != nil {
		return err
	}
	return r.ensureStatus(aerospikeCluster)
}
