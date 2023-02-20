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
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	"github.com/travelaudience/aerospike-operator/pkg/versioning"
)

const (
	clusterPrefix = "as-cluster-e2e-"
)

func (tf *TestFramework) NewAerospikeCluster(version string, nodeCount int32, namespaces []aerospikev1alpha2.AerospikeNamespaceSpec) aerospikev1alpha2.AerospikeCluster {
	return aerospikev1alpha2.AerospikeCluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: clusterPrefix,
		},
		Spec: aerospikev1alpha2.AerospikeClusterSpec{
			Version:    version,
			NodeCount:  nodeCount,
			Namespaces: namespaces,
		},
	}
}

func (tf *TestFramework) NewAerospikeClusterWithDefaults() aerospikev1alpha2.AerospikeCluster {
	aerospikeNamespace := tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1)
	latestVersion := versioning.AerospikeServerSupportedVersions[len(versioning.AerospikeServerSupportedVersions)-1]
	return tf.NewAerospikeCluster(latestVersion, 1, []aerospikev1alpha2.AerospikeNamespaceSpec{aerospikeNamespace})
}

func (tf *TestFramework) NewAerospikeNamespaceWithDeviceStorage(name string, replicationFactor int32, memorySizeGB int, defaultTTLSeconds int, storageSizeGB int) aerospikev1alpha2.AerospikeNamespaceSpec {
	return aerospikev1alpha2.AerospikeNamespaceSpec{
		Name:              name,
		ReplicationFactor: &replicationFactor,
		MemorySize:        pointers.NewString(fmt.Sprintf("%dG", memorySizeGB)),
		DefaultTTL:        pointers.NewString(fmt.Sprintf("%ds", defaultTTLSeconds)),
		Storage: aerospikev1alpha2.StorageSpec{
			Type: common.StorageTypeDevice,
			Size: fmt.Sprintf("%dG", storageSizeGB),
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceWithFileStorage(name string, replicationFactor int32, memorySizeGB int, defaultTTLSeconds int, storageSizeGB int) aerospikev1alpha2.AerospikeNamespaceSpec {
	return aerospikev1alpha2.AerospikeNamespaceSpec{
		Name:              name,
		ReplicationFactor: &replicationFactor,
		MemorySize:        pointers.NewString(fmt.Sprintf("%dG", memorySizeGB)),
		DefaultTTL:        pointers.NewString(fmt.Sprintf("%ds", defaultTTLSeconds)),
		Storage: aerospikev1alpha2.StorageSpec{
			Type: common.StorageTypeFile,
			Size: fmt.Sprintf("%dG", storageSizeGB),
		},
	}
}

func (tf *TestFramework) WaitForClusterCondition(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, fn watch.ConditionFunc, timeout time.Duration) error {
	fs := selectors.ObjectByCoordinates(aerospikeCluster.Namespace, aerospikeCluster.Name)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fs.String()
			return tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(aerospikeCluster.Namespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watchapi.Interface, error) {
			options.FieldSelector = fs.String()
			return tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(aerospikeCluster.Namespace).Watch(context.TODO(), options)
		},
	}
	ctx, cfn := context.WithTimeout(context.Background(), timeout)
	defer cfn()
	last, err := watch.UntilWithSync(ctx, lw, &aerospikev1alpha2.AerospikeCluster{}, nil, fn)
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for aerospikecluster %q", meta.Key(aerospikeCluster))
	}
	return nil
}

func (tf *TestFramework) WaitForClusterNodeCountOrTimeout(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, nodeCount int32, timeout time.Duration) error {
	return tf.WaitForClusterCondition(aerospikeCluster, func(event watchapi.Event) (bool, error) {
		// grab the current cluster object from the event
		obj := event.Object.(*aerospikev1alpha2.AerospikeCluster)
		// search for the current node count
		return obj.Status.NodeCount == nodeCount, nil
	}, timeout)
}

func (tf *TestFramework) WaitForClusterNodeCount(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, nodeCount int32) error {
	return tf.WaitForClusterNodeCountOrTimeout(aerospikeCluster, nodeCount, watchTimeout)
}

func (tf *TestFramework) ScaleCluster(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, nodeCount int32) error {
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(aerospikeCluster.Namespace).Get(context.TODO(), aerospikeCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	res.Spec.NodeCount = nodeCount
	res, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(res.Namespace).Update(context.TODO(), res, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return tf.WaitForClusterNodeCount(res, nodeCount)
}

func (tf *TestFramework) ChangeNamespaceMemorySizeAndScaleClusterAndWait(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, newMemorySizeGB int, nodeCount int32) error {
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(aerospikeCluster.Namespace).Get(context.TODO(), aerospikeCluster.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	res.Spec.Namespaces[0].MemorySize = pointers.NewString(fmt.Sprintf("%dG", newMemorySizeGB))
	res.Spec.NodeCount = nodeCount
	if _, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(res.Namespace).Update(context.TODO(), res, metav1.UpdateOptions{}); err != nil {
		return err
	}
	return tf.WaitForClusterNodeCount(res, nodeCount)
}

func (tf *TestFramework) NewAerospikeClusterV1alpha1(version string, nodeCount int32, namespaces []aerospikev1alpha1.AerospikeNamespaceSpec) aerospikev1alpha1.AerospikeCluster {
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

func (tf *TestFramework) NewV1alpha1AerospikeClusterWithDefaults() aerospikev1alpha1.AerospikeCluster {
	aerospikeNamespace := tf.NewAerospikeNamespaceWithFileStorageV1alpha1("aerospike-namespace-0", 1, 1, 0, 1)
	latestVersion := versioning.AerospikeServerSupportedVersions[len(versioning.AerospikeServerSupportedVersions)-1]
	return tf.NewAerospikeClusterV1alpha1(latestVersion, 1, []aerospikev1alpha1.AerospikeNamespaceSpec{aerospikeNamespace})
}

func (tf *TestFramework) NewAerospikeNamespaceWithFileStorageV1alpha1(name string, replicationFactor int32, memorySizeGB int, defaultTTLSeconds int, storageSizeGB int) aerospikev1alpha1.AerospikeNamespaceSpec {
	return aerospikev1alpha1.AerospikeNamespaceSpec{
		Name:              name,
		ReplicationFactor: &replicationFactor,
		MemorySize:        pointers.NewString(fmt.Sprintf("%dG", memorySizeGB)),
		DefaultTTL:        pointers.NewString(fmt.Sprintf("%ds", defaultTTLSeconds)),
		Storage: aerospikev1alpha1.StorageSpec{
			Type: common.StorageTypeFile,
			Size: fmt.Sprintf("%dG", storageSizeGB),
		},
	}
}
