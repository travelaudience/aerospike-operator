package reconciler

import (
	"k8s.io/api/core/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
)

func (r *AerospikeClusterReconciler) validate(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (bool, error) {
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonValidationError,
				"replication factor of %d requested for namespace %s but the cluster has only %d nodes",
				ns.ReplicationFactor,
				ns.Name,
				aerospikeCluster.Spec.NodeCount,
			)
			return false, nil
		}
	}
	return true, nil
}
