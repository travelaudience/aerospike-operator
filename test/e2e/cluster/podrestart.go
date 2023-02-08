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

package cluster

import (
	"context"
	"fmt"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/asutils"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testNodeCountAfterRestart(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	err = tf.ChangeNamespaceMemorySizeAndScaleClusterAndWait(asc, 4, nodeCount)

	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(asc.Namespace).Get(context.TODO(), asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.Status.NodeCount).To(Equal(nodeCount))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(nodeCount))
}

func testNodeCountAfterRestartAndScaling(tf *framework.TestFramework, ns *v1.Namespace, initialNodeCount, finalNodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = initialNodeCount
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, initialNodeCount)
	Expect(err).NotTo(HaveOccurred())

	err = tf.ChangeNamespaceMemorySizeAndScaleClusterAndWait(asc, 4, finalNodeCount)
	Expect(err).NotTo(HaveOccurred())

	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(asc.Namespace).Get(context.TODO(), asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.Status.NodeCount).To(Equal(finalNodeCount))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(finalNodeCount))
}

func testNoDataLossAfterRestart(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32, nRecords int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	aerospikeCluster.Spec.Namespaces[0].ReplicationFactor = pointers.NewInt32(2)
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(res)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	err = tf.ChangeNamespaceMemorySizeAndScaleClusterAndWait(res, 4, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	c2, err := framework.NewAerospikeClient(res)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()
}

func testNoDataLossAfterRestartAndScaleDown(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32, nRecords int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount + 1
	aerospikeCluster.Spec.Namespaces[0].ReplicationFactor = pointers.NewInt32(2)
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, nodeCount+1)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	err = tf.ChangeNamespaceMemorySizeAndScaleClusterAndWait(asc, 4, nodeCount+1)
	Expect(err).NotTo(HaveOccurred())

	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(asc.Namespace).Get(context.TODO(), asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	err = tf.ScaleCluster(asc, nodeCount)

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(nodeCount))

	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()
}

func testNodeIDsAfterRestart(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	nodeNamesBeforeRestart := c1.GetNodeNames()
	c1.Close()

	err = tf.ChangeNamespaceMemorySizeAndScaleClusterAndWait(asc, 4, nodeCount)

	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	nodeNamesAfterRestart := c2.GetNodeNames()
	c2.Close()

	for _, node := range nodeNamesBeforeRestart {
		found := false
		for _, newNode := range nodeNamesAfterRestart {
			if node == newNode {
				found = true
			}
		}
		Expect(found).To(Equal(true))
	}
}
