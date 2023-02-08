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

package framework

import (
	"context"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
)

// UpgradeClusterAndWait upgrades an Aerospike cluster to the specified targetVersion
func (tf *TestFramework) UpgradeClusterAndWait(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, targetVersion string) (*aerospikev1alpha2.AerospikeCluster, error) {
	// change Aerospike cluster version
	aerospikeCluster.Spec.Version = targetVersion
	// update the Aerospike cluster
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(aerospikeCluster.Namespace).Update(context.TODO(), aerospikeCluster, v1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	// wait for .status.version to be equal to targetVersion
	return asc, tf.WaitForClusterCondition(asc, func(event watch.Event) (bool, error) {
		// grab the current cluster object from the event
		asc = event.Object.(*aerospikev1alpha2.AerospikeCluster)
		// check if .status.version is equal to targetVersion
		return asc.Status.Version == targetVersion, nil
	}, watchTimeout)
}
