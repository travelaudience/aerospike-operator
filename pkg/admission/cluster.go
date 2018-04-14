package admission

import (
	"fmt"
	"reflect"

	av1beta1 "k8s.io/api/admission/v1beta1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
)

func admitAerospikeCluster(ar av1beta1.AdmissionReview) *av1beta1.AdmissionResponse {
	// decode the new AerospikeCluster object
	new, err := decodeAerospikeCluster(ar.Request.Object.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}
	// decode the old AerospikeCluster object (if any)
	old, err := decodeAerospikeCluster(ar.Request.OldObject.Raw)
	if err != nil {
		return admissionResponseFromError(err)
	}
	// validate the new AerospikeCluster
	if err = validateAerospikeCluster(new); err != nil {
		return admissionResponseFromError(err)
	}
	// if this is an update, validate that the transition from old to new
	if ar.Request.Operation == av1beta1.Update {
		if err = validateAerospikeClusterUpdate(old, new); err != nil {
			return admissionResponseFromError(err)
		}
	}
	// admit the AerospikeCluster object
	return &av1beta1.AdmissionResponse{Allowed: true}
}

func validateAerospikeCluster(aerospikeCluster *v1alpha1.AerospikeCluster) error {
	// validate that every namespace's replication factor is less than or equal to the cluster's node count.
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.ReplicationFactor > aerospikeCluster.Spec.NodeCount {
			return fmt.Errorf("replication factor of %d requested for namespace %s but the cluster has only %d nodes", ns.ReplicationFactor, ns.Name, aerospikeCluster.Spec.NodeCount)
		}
	}
	return nil
}

func validateAerospikeClusterUpdate(old, new *v1alpha1.AerospikeCluster) error {
	// grab a name => spec map for the namespaces in the old object
	oldnss := namespaceMap(old)
	// grab a name => spec map for the namespaces in the new object
	newnss := namespaceMap(new)
	// validate that no namespace has been removed
	for name := range oldnss {
		if _, ok := newnss[name]; !ok {
			return fmt.Errorf("cannot remove namespace %s", name)
		}
	}
	// validate that there were no changes to existing namespaces
	for name := range newnss {
		if _, ok := oldnss[name]; !ok {
			continue
		}
		if !reflect.DeepEqual(oldnss[name], newnss[name]) {
			return fmt.Errorf("cannot change the spec for namespace %s", name)
		}
	}
	return nil
}

func namespaceMap(aerospikeCluster *v1alpha1.AerospikeCluster) map[string]v1alpha1.AerospikeNamespaceSpec {
	res := make(map[string]v1alpha1.AerospikeNamespaceSpec, len(aerospikeCluster.Spec.Namespaces))
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		res[ns.Name] = ns
	}
	return res
}

func decodeAerospikeCluster(raw []byte) (*v1alpha1.AerospikeCluster, error) {
	obj := &v1alpha1.AerospikeCluster{}
	if len(raw) == 0 {
		return obj, nil
	}
	_, _, err := codecs.UniversalDeserializer().Decode(raw, nil, obj)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
