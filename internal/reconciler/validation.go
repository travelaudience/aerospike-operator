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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/utils/events"
)

func (r *AerospikeClusterReconciler) validate(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (bool, error) {
	if !r.validateReplicationFactor(aerospikeCluster) {
		return false, nil
	}
	if !r.validateStorageClass(aerospikeCluster) {
		return false, nil
	}
	return true, nil
}

func (r *AerospikeClusterReconciler) validateReplicationFactor(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) bool {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.ReplicationFactor != nil && *ns.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			r.recorder.Eventf(aerospikeCluster, corev1.EventTypeWarning, events.ReasonValidationError,
				"replication factor of %d requested for namespace %s but the cluster has only %d nodes",
				ns.ReplicationFactor,
				ns.Name,
				aerospikeCluster.Spec.NodeCount,
			)
			return false
		}
	}
	return true
}

func (r *AerospikeClusterReconciler) validateStorageClass(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) bool {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.Storage.StorageClassName != nil && *ns.Storage.StorageClassName != "" {
			if _, err := r.scsLister.Get(*ns.Storage.StorageClassName); err != nil {
				if errors.IsNotFound(err) {
					r.recorder.Eventf(aerospikeCluster, corev1.EventTypeWarning, events.ReasonValidationError,
						"storage class %q does not exist",
						ns.Storage.StorageClassName,
					)
				} else {
					r.recorder.Eventf(aerospikeCluster, corev1.EventTypeWarning, events.ReasonValidationError,
						"failed to get storage class %q: %v",
						ns.Storage.StorageClassName,
						err,
					)
				}
				return false
			}
		}
	}
	return true
}
