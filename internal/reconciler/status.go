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
	"reflect"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/logfields"
	"github.com/travelaudience/aerospike-operator/internal/meta"
)

// updateStatus updates the status of aerospikeCluster to match the spec.
// IMPORTANT this method MUST only be called after a successful reconcile
func (r *AerospikeClusterReconciler) updateStatus(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) {
	// update status to match the spec - the correctness of this is ensured by
	// the reconcile loop
	aerospikeCluster.Status.BackupSpec = aerospikeCluster.Spec.BackupSpec
	aerospikeCluster.Status.Namespaces = aerospikeCluster.Spec.Namespaces
	aerospikeCluster.Status.NodeCount = aerospikeCluster.Spec.NodeCount
	aerospikeCluster.Status.Version = aerospikeCluster.Spec.Version
}

// patchCluster updates the aerospikecluster resource.
func (r *AerospikeClusterReconciler) patchCluster(old, new *aerospikev1alpha2.AerospikeCluster) error {
	// return if there are no changes to patch
	if reflect.DeepEqual(old, new) {
		return nil
	}
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return err
	}
	newBytes, err := json.Marshal(new)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, &aerospikev1alpha2.AerospikeCluster{})
	if err != nil {
		return err
	}
	// grab the status changes before patching
	newStatus := new.Status
	new, err = r.aerospikeclientset.AerospikeV1alpha2().AerospikeClusters(old.Namespace).Patch(old.Name, types.MergePatchType, patchBytes)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(new),
	}).Debug("resource updated")

	// update the status subresource
	if !reflect.DeepEqual(new.Status, newStatus) {
		new.Status = newStatus
		new, err = r.aerospikeclientset.AerospikeV1alpha2().AerospikeClusters(new.Namespace).UpdateStatus(new)
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(new),
		}).Debug("status updated")
	}
	return nil
}

// appendCondition appends the specified condition to the aerospikeCluster
// object
func appendCondition(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, condition apiextensions.CustomResourceDefinitionCondition) {
	aerospikeCluster.Status.Conditions = append(aerospikeCluster.Status.Conditions, condition)
}

// setAerospikeClusterAnnotation sets an annotation with the specified key and value in the
// aerospikecluster object
func setAerospikeClusterAnnotation(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, key, value string) {
	if aerospikeCluster.ObjectMeta.Annotations == nil {
		aerospikeCluster.ObjectMeta.Annotations = make(map[string]string)
	}
	aerospikeCluster.ObjectMeta.Annotations[key] = value
}

// removeAerospikeClusterAnnotation removes the annotation with the specified key from the
// aerospikecluster object
func removeAerospikeClusterAnnotation(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, key string) {
	delete(aerospikeCluster.ObjectMeta.Annotations, key)
}
