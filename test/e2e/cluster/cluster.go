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
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/travelaudience/aerospike-operator/pkg/admission"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/asutils"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testCreateAerospikeClusterWithLengthyName(tf *framework.TestFramework, ns *corev1.Namespace) {
	// create the name of the cluster by appending as many 'a' runes as necessary in order to exceed the limit
	var sb strings.Builder
	for sb.Len() <= admission.AerospikeClusterNameMaxLength {
		sb.WriteRune('a')
	}
	// create the cluster and make sure we've got the expected error as a result
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Name = sb.String()
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	status := err.(*errors.StatusError)
	Expect(status.ErrStatus.Status).To(Equal(metav1.StatusFailure))
	Expect(status.ErrStatus.Message).To(MatchRegexp("the name of the cluster cannot exceed 61 characters"))
}

func testCreateAerospikeClusterWithLengthyNameAndNamespace(tf *framework.TestFramework, ns *corev1.Namespace) {
	// create the name of the cluster by appending as many 'a' runes as necessary in order to exceed the limit
	var sb strings.Builder
	for 2*(sb.Len()+2)+len(ns.Name) <= admission.AerospikeMeshSeedAddressMaxLength {
		sb.WriteRune('a')
	}
	// create the cluster and make sure we've got the expected error as a result
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Name = sb.String()
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	status := err.(*errors.StatusError)
	Expect(status.ErrStatus.Status).To(Equal(metav1.StatusFailure))
	Expect(status.ErrStatus.Message).To(MatchRegexp("the current combination of cluster and kubernetes namespace names cannot be used"))
}

func testCreateAerospikeClusterWithZeroNodes(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = 0
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.nodeCount.*should be greater than or equal to 1")))
}

func testCreateAerospikeClusterWithNineNodes(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = 9
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(errors.IsInvalid(err)).To(BeTrue())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("spec.nodeCount.*should be less than or equal to 8")))
}

func testCreateAerospikeClusterWithZeroNamespaces(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []aerospikev1alpha2.AerospikeNamespaceSpec{}
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("the number of namespaces in the cluster must be exactly one")))
}

func testCreateAerospikeClusterWithTwoNamespaces(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []aerospikev1alpha2.AerospikeNamespaceSpec{
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1),
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-1", 1, 1, 0, 1),
	}
	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	Expect(tf.ErrorCauses(err)).To(ContainElement(MatchRegexp("the number of namespaces in the cluster must be exactly one")))
}

func testCreateAerospikeClusterWithInvalidReplicationFactor(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces = []aerospikev1alpha2.AerospikeNamespaceSpec{
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", aerospikeCluster.Spec.NodeCount+1, 1, 0, 1),
	}

	_, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).To(HaveOccurred())
	status := err.(*errors.StatusError)
	Expect(status.ErrStatus.Status).To(Equal(metav1.StatusFailure))
	Expect(status.ErrStatus.Message).To(MatchRegexp("replication factor of \\d+ requested for namespace .+ but the cluster has only \\d+ nodes"))
}

func testCreateAerospikeClusterWithNodeCount(tf *framework.TestFramework, ns *corev1.Namespace, nodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(nodeCount))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", res.Name, res.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(nodeCount))
}

func testCreateAerospikeClusterWithResources(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Resources = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("1212Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1500m"),
			corev1.ResourceMemory: resource.MustParse("1512Mi"),
		},
	}

	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, 1)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(int32(1)))
	Expect(pods.Items[0].Spec.Containers[0].Resources.Requests.Cpu()).To(Equal(aerospikeCluster.Spec.Resources.Requests.Cpu()))
	Expect(pods.Items[0].Spec.Containers[0].Resources.Requests.Memory()).To(Equal(aerospikeCluster.Spec.Resources.Requests.Memory()))
	Expect(pods.Items[0].Spec.Containers[0].Resources.Limits.Cpu()).To(Equal(aerospikeCluster.Spec.Resources.Limits.Cpu()))
	Expect(pods.Items[0].Spec.Containers[0].Resources.Limits.Memory()).To(Equal(aerospikeCluster.Spec.Resources.Limits.Memory()))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", res.Name, res.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(int32(1)))
}

func testCreateAerospikeWithComputedResources(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Resources = &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("212Mi"),
		},
	}

	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, 1)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(int32(1)))
	Expect(pods.Items[0].Spec.Containers[0].Resources.Requests.Cpu()).NotTo(BeNil())
	Expect(pods.Items[0].Spec.Containers[0].Resources.Requests.Memory()).NotTo(Equal(aerospikeCluster.Spec.Resources.Requests.Memory()))
}

func testConnectToAerospikeCluster(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, res.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asc, err := framework.NewAerospikeClient(res)
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.IsConnected()).To(BeTrue())
}

func testCreateAerospikeClusterWithV1alpha1(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewV1alpha1AerospikeClusterWithDefaults()
	_, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())
}

func testDataInMemory(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces[0].Storage.DataInMemory = pointers.NewBool(true)
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	c, err := framework.NewAerospikeClient(res)
	Expect(err).NotTo(HaveOccurred())

	dataInMemoryEnabled, err := c.IsDataInMemoryEnabled(aerospikeCluster.Spec.Namespaces[0].Name)
	Expect(err).NotTo(HaveOccurred())
	Expect(dataInMemoryEnabled).To(Equal(true))
}

func testCreateAerospikeWithNodeSelector(tf *framework.TestFramework, ns *corev1.Namespace) {

	nodes, err := tf.KubeClient.CoreV1().Nodes().List(metav1.ListOptions{})

	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeSelector = nodes.Items[0].Labels

	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, 1)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(int32(1)))
}

func testCreateAerospikeWithInvalidNodeSelector(tf *framework.TestFramework, ns *corev1.Namespace) {

	randomLabels := map[string]string{
		"random": "label",
	}

	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeSelector = randomLabels

	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCountOrTimeout(res, 1, time.Minute)
	Expect(err).To(HaveOccurred())
}

func testCreateAerospikeWithTolerations(tf *framework.TestFramework, ns *corev1.Namespace) {

	tolerations := []corev1.Toleration{{Key: "nodetype", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}}

	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Tolerations = tolerations

	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, 1)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(len(pods.Items))).To(Equal(int32(1)))
	Expect(pods.Items[0].Spec.Tolerations).To(ContainElement(tolerations[0]))
}

func testAerospikeNodeFail(tf *framework.TestFramework, ns *corev1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = 3
	res, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	tf.WaitForClusterNodeCount(res, 3)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.ResourcesByClusterName(res.Name))
	command := []string{"/bin/sh", "-c", "kill 1"}
	pod := pods.Items[2]
	tf.ExecInContainer(pod, ns, command, "asprom")

	err = tf.WaitForClusterNodeCountOrTimeout(res, 3, time.Minute)
	Expect(err).NotTo(HaveOccurred())

	podState, err := tf.KubeClient.CoreV1().Pods(ns.Name).Get(pod.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(podState.Status).NotTo(Equal(metav1.StatusFailure))
	Expect(podState.Status.Phase).To(Equal(corev1.PodRunning))
}

