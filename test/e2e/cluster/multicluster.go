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
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/asutils"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testClusterSizeAfterScalingDownClusterWhileStartingAnother(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int) {
	aerospikeCluster1 := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster1.Spec.NodeCount = nodeCount + 1
	asc1, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster1)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc1, nodeCount+1)
	Expect(err).NotTo(HaveOccurred())

	err = tf.ScaleCluster(asc1, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	aerospikeCluster2 := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster2.Spec.NodeCount = nodeCount
	asc2, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster2)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc2, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	asc1, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc1.Namespace).Get(asc1.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc1.Status.NodeCount).To(Equal(nodeCount))

	asc2, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc2.Namespace).Get(asc2.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc2.Status.NodeCount).To(Equal(nodeCount))

	svc1, err := tf.CreateNodePortService(asc1)
	Expect(err).NotTo(HaveOccurred())
	err = tf.WaitForNodePortService(svc1)
	Expect(err).NotTo(HaveOccurred())

	svc2, err := tf.CreateNodePortService(asc2)
	Expect(err).NotTo(HaveOccurred())
	err = tf.WaitForNodePortService(svc2)
	Expect(err).NotTo(HaveOccurred())

	size1, err := asutils.GetClusterSize(framework.NodeAddress, int(svc1.Spec.Ports[0].NodePort))
	Expect(err).NotTo(HaveOccurred())
	Expect(size1).To(Equal(nodeCount))

	size2, err := asutils.GetClusterSize(framework.NodeAddress, int(svc2.Spec.Ports[0].NodePort))
	Expect(err).NotTo(HaveOccurred())
	Expect(size2).To(Equal(nodeCount))
}
