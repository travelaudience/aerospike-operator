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
	"math"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/internal/asutils"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testNodeCountAfterScaling(tf *framework.TestFramework, ns *v1.Namespace, initialNodeCount, finalNodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = initialNodeCount
	asc, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, initialNodeCount)
	Expect(err).NotTo(HaveOccurred())

	err = tf.ScaleCluster(asc, finalNodeCount)
	Expect(err).NotTo(HaveOccurred())

	asc, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc.Namespace).Get(asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.Status.NodeCount).To(Equal(finalNodeCount))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(finalNodeCount))
}

func testNoDowntimeDuringScaling(tf *framework.TestFramework, ns *v1.Namespace, initialNodeCount, finalNodeCount int32) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = initialNodeCount
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{
		tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", int32(math.Min(float64(initialNodeCount), float64(finalNodeCount))), 1, 0, 1),
	}
	asc, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())
	err = tf.WaitForClusterNodeCount(asc, initialNodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(asc.Spec.Namespaces[0].Name, 1000)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	// cerrCh will contain any connection errors that may occur
	cerrCh := make(chan error)
	// stopCh will allow us to exit the goroutine
	stopCh := make(chan bool)
	// keep reading integer values until cluster finishes scaling
	go func() {
		defer close(cerrCh)
		for {
			select {
			case <-stopCh:
				return
			default:
				c2, err := framework.NewAerospikeClient(asc)
				Expect(err).NotTo(HaveOccurred())
				if err = c2.ReadSequentialIntegers(asc.Spec.Namespaces[0].Name, 1000); err != nil {
					cerrCh <- err
				}
				c2.Close()
			}
		}
	}()

	err = tf.ScaleCluster(asc, finalNodeCount)
	Expect(err).NotTo(HaveOccurred())

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(finalNodeCount))

	close(stopCh)
	c1.Close()

	// expect no errors in cerrCh
	for err := range cerrCh {
		Expect(err).NotTo(HaveOccurred())
	}
}
