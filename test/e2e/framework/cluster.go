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

package framework

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
)

const (
	clusterPrefix = "as-cluster-e2e-"
)

func (tf *TestFramework) NewAerospikeCluster(version string, nodeCount int32, namespaces []aerospikev1alpha1.AerospikeNamespaceSpec) aerospikev1alpha1.AerospikeCluster {
	return aerospikev1alpha1.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: clusterPrefix,
		},
		Spec: aerospikev1alpha1.AerospikeClusterSpec{
			Version:    version,
			NodeCount:  nodeCount,
			Namespaces: namespaces,
		},
	}
}

func (tf *TestFramework) NewAerospikeClusterWithDefaults() aerospikev1alpha1.AerospikeCluster {
	aerospikeNamespace := tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1)
	return tf.NewAerospikeCluster("4.2.0.3", 1, []aerospikev1alpha1.AerospikeNamespaceSpec{aerospikeNamespace})
}

func (tf *TestFramework) NewAerospikeNamespaceWithFileStorage(name string, replicationFactor int32, memorySizeGB int, defaultTTLSeconds int, storageSizeGB int) aerospikev1alpha1.AerospikeNamespaceSpec {
	return aerospikev1alpha1.AerospikeNamespaceSpec{
		Name:              name,
		ReplicationFactor: &replicationFactor,
		MemorySize:        pointers.NewString(fmt.Sprintf("%dG", memorySizeGB)),
		DefaultTTL:        pointers.NewString(fmt.Sprintf("%ds", defaultTTLSeconds)),
		Storage: aerospikev1alpha1.StorageSpec{
			Type: aerospikev1alpha1.StorageTypeFile,
			Size: fmt.Sprintf("%dG", storageSizeGB),
		},
	}
}

func (tf *TestFramework) WaitForClusterCondition(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, fn watch.ConditionFunc, timeout time.Duration) error {
	w, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(aerospikeCluster.Namespace).Watch(listoptions.ObjectByName(aerospikeCluster.Name))
	if err != nil {
		return err
	}
	start := time.Now()
	last, err := watch.Until(timeout, w, fn)
	if err != nil {
		if err == watch.ErrWatchClosed {
			if t := timeout - time.Since(start); t > 0 {
				return tf.WaitForClusterCondition(aerospikeCluster, fn, t)
			}
		}
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for %s", meta.Key(aerospikeCluster))
	}
	return nil
}

func (tf *TestFramework) WaitForClusterNodeCount(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, nodeCount int32) error {
	return tf.WaitForClusterCondition(aerospikeCluster, func(event watch.Event) (bool, error) {
		// grab the current cluster object from the event
		obj := event.Object.(*aerospikev1alpha1.AerospikeCluster)
		// search for the current node count
		return obj.Status.NodeCount == nodeCount, nil
	}, watchTimeout)
}

func (tf *TestFramework) ScaleCluster(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, nodeCount int32) error {
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(aerospikeCluster.Namespace).Get(aerospikeCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	res.Spec.NodeCount = nodeCount
	res, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(res.Namespace).Update(res)
	if err != nil {
		return err
	}
	return tf.WaitForClusterNodeCount(res, nodeCount)
}

func (tf *TestFramework) ChangeNamespaceMemorySizeAndScaleClusterAndWait(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, newMemorySizeGB int, nodeCount int32) error {
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(aerospikeCluster.Namespace).Get(aerospikeCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	res.Spec.Namespaces[0].MemorySize = pointers.NewString(fmt.Sprintf("%dG", newMemorySizeGB))
	res.Spec.NodeCount = nodeCount
	if _, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(res.Namespace).Update(res); err != nil {
		return err
	}
	return tf.WaitForClusterCondition(res, func(event watch.Event) (bool, error) {
		obj := event.Object.(*aerospikev1alpha1.AerospikeCluster)
		return len(obj.Status.Namespaces) == len(obj.Spec.Namespaces) && obj.Status.NodeCount == nodeCount, nil
	}, watchTimeout)
}
