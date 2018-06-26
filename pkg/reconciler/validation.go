package reconciler

import (
	"k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/errors"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
)

func (r *AerospikeClusterReconciler) validate(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (bool, error) {
	if !r.validateReplicationFactor(aerospikeCluster) {
		return false, nil
	}
	if !r.validateStorageClass(aerospikeCluster) {
		return false, nil
	}
	return true, nil
}

func (r *AerospikeClusterReconciler) validateReplicationFactor(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) bool {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.ReplicationFactor != nil && *ns.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonValidationError,
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

func (r *AerospikeClusterReconciler) validateStorageClass(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) bool {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.Storage.StorageClassName != "" {
			if _, err := r.scsLister.Get(ns.Storage.StorageClassName); err != nil {
				if errors.IsNotFound(err) {
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonValidationError,
						"storage class %q does not exist",
						ns.Storage.StorageClassName,
					)
				} else {
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonValidationError,
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
