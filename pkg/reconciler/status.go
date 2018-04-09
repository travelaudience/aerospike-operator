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
	"encoding/json"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
)

// ensureStatus observes the aerospikeCluster and updates its status field accordingly
func (r *AerospikeClusterReconciler) ensureStatus(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	// deepcopy aerospikeCluster so we can modify the status to later create the patch
	new := aerospikeCluster.DeepCopy()
	// lookup pods belonging to this aerospikeCluster
	pods, err := r.kubeclientset.CoreV1().Pods(aerospikeCluster.Namespace).List(listoptions.PodsByClusterName(aerospikeCluster.Name))
	if err != nil {
		return err
	}
	// update status.nodeCount accordingly
	new.Status.NodeCount = len(pods.Items)
	// update status.version and status.namespaces to match spec
	new.Status.Version = aerospikeCluster.Spec.Version
	new.Status.Namespaces = aerospikeCluster.Spec.Namespaces
	// update the status
	return r.updateStatus(aerospikeCluster, new)
}

// updateStatus updates the status field of the aerospikeCluster
func (r *AerospikeClusterReconciler) updateStatus(old, new *aerospikev1alpha1.AerospikeCluster) error {
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
	return nil
}
