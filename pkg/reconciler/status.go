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
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

// updateStatus updates the status of aerospikeCluster to match the spec.
// IMPORTANT this method MUST only be called after a successful reconcile
func (r *AerospikeClusterReconciler) updateStatus(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	// deepcopy aerospikeCluster so we can modify the status to later create the patch
	new := aerospikeCluster.DeepCopy()
	// update status to match the spec - the correctness of this is ensured by
	// the reconcile loop
	new.Status.Namespaces = aerospikeCluster.Spec.Namespaces
	new.Status.NodeCount = aerospikeCluster.Spec.NodeCount
	new.Status.Version = aerospikeCluster.Spec.Version
	// update the status field of aerospikeCluster
	return r.patchCluster(aerospikeCluster, new)
}

// patchCluster updates the status field of the aerospikeCluster
func (r *AerospikeClusterReconciler) patchCluster(old, new *aerospikev1alpha1.AerospikeCluster) error {
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return err
	}
	newBytes, err := json.Marshal(new)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, &aerospikev1alpha1.AerospikeCluster{})
	if err != nil {
		return err
	}
	_, err = r.aerospikeclientset.AerospikeV1alpha1().AerospikeClusters(old.Namespace).Patch(old.Name, types.MergePatchType, patchBytes)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(new),
	}).Debug("status updated")
	return nil
}

// appendCondition appends the specified condition to the aerospikeCluster
// object
func appendCondition(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, condition apiextensions.CustomResourceDefinitionCondition) {
	aerospikeCluster.Status.Conditions = append(aerospikeCluster.Status.Conditions, condition)
}

// setAnnotation sets an annotation with the specified key and value in the
// aerospikeCluster object
func setAnnotation(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, key, value string) {
	if aerospikeCluster.ObjectMeta.Annotations == nil {
		aerospikeCluster.ObjectMeta.Annotations = make(map[string]string)
	}
	aerospikeCluster.ObjectMeta.Annotations[key] = value
}

// removeAnnotation removes the annotation with the specified key from the
// aerospikeCluster object
func removeAnnotation(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, key string) {
	delete(aerospikeCluster.ObjectMeta.Annotations, key)
}
