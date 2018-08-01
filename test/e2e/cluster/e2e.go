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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"

	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

var (
	tf *framework.TestFramework
)

func RegisterTestFramework(testFramework *framework.TestFramework) {
	tf = testFramework
}

var _ = Describe("AerospikeCluster", func() {
	var (
		ns *v1.Namespace
	)

	Context("in dedicated namespace", func() {
		BeforeEach(func() {
			var err error
			ns, err = tf.CreateRandomNamespace()
			Expect(err).NotTo(HaveOccurred())
			err = tf.CopySecretToNamespace(framework.GCSSecretName, ns)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := tf.DeleteNamespace(ns)
			Expect(err).NotTo(HaveOccurred())
		})

		It("cannot be created with len(metadata.name)==61", func() {
			testCreateAerospikeClusterWithLengthyName(tf, ns)
		})
		It("cannot be created with spec.nodeCount==0", func() {
			testCreateAerospikeClusterWithZeroNodes(tf, ns)
		})
		It("cannot be created with spec.nodeCount==9", func() {
			testCreateAerospikeClusterWithNineNodes(tf, ns)
		})
		It("cannot be created with len(spec.namespaces)==0", func() {
			testCreateAerospikeClusterWithZeroNamespaces(tf, ns)
		})
		It("cannot be created with len(spec.namespaces)==2", func() {
			testCreateAerospikeClusterWithTwoNamespaces(tf, ns)
		})
		It("cannot be created if spec.namespaces.replicationFactor[*] > spec.nodeCount", func() {
			testCreateAerospikeClusterWithInvalidReplicationFactor(tf, ns)
		})
		It("is created with the provided spec.nodeCount", func() {
			testCreateAerospikeClusterWithNodeCount(tf, ns, 2)
		})
		It("accepts connections on the service port", func() {
			testConnectToAerospikeCluster(tf, ns)
		})
		It("supports device storage", func() {
			testDeviceStorage(tf, ns, 2, 10)
		})
		It("supports file storage", func() {
			testFileStorage(tf, ns, 1, 2)
		})
		It("reuses the persistent volume of a deleted pod", func() {
			testVolumeIsReused(tf, ns, 2)
		})
		It("has the correct number of nodes after scaling up", func() {
			testNodeCountAfterScaling(tf, ns, 1, 3)
		})
		It("has the correct number of nodes after scaling down", func() {
			testNodeCountAfterScaling(tf, ns, 3, 1)
		})
		It("has the same number of nodes after rolling restart", func() {
			testNodeCountAfterRestart(tf, ns, 2)
		})
		It("has the correct number of nodes after rolling restart and scaling up", func() {
			testNodeCountAfterRestartAndScaling(tf, ns, 2, 3)
		})
		It("has the correct number of nodes after rolling restart and scaling down", func() {
			testNodeCountAfterRestartAndScaling(tf, ns, 3, 1)
		})
		It("does not lose data in a namespace after rolling restart", func() {
			testNoDataLossAfterRestart(tf, ns, 2, 10000)
		})
		It("does not lose data in a namespace after rolling restart and scale down", func() {
			testNoDataLossAfterRestartAndScaleDown(tf, ns, 2, 100000)
		})
		It("has no downtime during a scale up operation", func() {
			testNoDowntimeDuringScaling(tf, ns, 2, 3)
		})
		It("has no downtime during a scale down operation", func() {
			testNoDowntimeDuringScaling(tf, ns, 3, 2)
		})
		It("does not conflict with another AerospikeCluster", func() {
			testClusterSizeAfterScalingDownClusterWhileStartingAnother(tf, ns, 1)
		})
		It("does not lose data in a namespace after an upgrade and makes automatic backup of namespaces before upgrade", func() {
			testNoDataLossOnAerospikeUpgrade(tf, ns, 2, 10000, "4.2.0.3", "4.2.0.4")
		})
		It("node IDs are kept after restart", func() {
			testNodeIDsAfterRestart(tf, ns, 2)
		})
	})
})
