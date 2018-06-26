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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testCreateAerospikeClusterWithLengthyName(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Name = "one-really-lengthy-cluster-name-having-more-than-sixty-charss"
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	status := err.(*errors.StatusError)
	Expect(status.ErrStatus.Status).To(Equal(metav1.StatusFailure))
	Expect(status.ErrStatus.Message).To(MatchRegexp("the name of the cluster cannot exceed 60 characters"))
}

func testCreateAerospikeClusterWithZeroNodes(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = 0
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.nodeCount.*should be greater than or equal to 1")))
}

func testCreateAerospikeClusterWithNineNodes(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = 9
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.nodeCount.*should be less than or equal to 8")))
}

func testCreateAerospikeClusterWithZeroNamespaces(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{}
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.namespaces.*should have at least 1 items")))
}

func testCreateAerospikeClusterWithThreeNamespaces(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1),
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-1", 1, 1, 0, 1),
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-2", 1, 1, 0, 1),
	}
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.namespaces.*should have at most 2 items")))
}

func testCreateAerospikeClusterWithInvalidReplicationFactor(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", aerospikeCluster.Spec.NodeCount+1, 1, 0, 1),
	}

	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	status := err.(*errors.StatusError)
	Expect(status.ErrStatus.Status).To(Equal(metav1.StatusFailure))
	Expect(status.ErrStatus.Message).To(MatchRegexp("replication factor of \\d+ requested for namespace .+ but the cluster has only \\d+ nodes"))
}

func testCreateAerospikeClusterWithNodeCount(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.PodsByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(nodeCount))
}

func testConnectToAerospikeCluster(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, res.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asc, err := framework.NewAerospikeClient(res)
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.IsConnected()).To(BeTrue())
}
