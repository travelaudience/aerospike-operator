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

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testVolumesSizeMatchNamespaceSpec(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int, nsSize1 int, nsSize2 int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	ns1 := tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, nsSize1)
	ns2 := tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-1", 1, 1, 0, nsSize2)
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{ns1, ns2}
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.PodsByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(len(pods.Items)).To(Equal(nodeCount))

	for _, pod := range pods.Items {
		Expect(pod.Spec.Volumes).NotTo(BeEmpty())
		for _, volume := range pod.Spec.Volumes {
			if volume.VolumeSource.PersistentVolumeClaim != nil {
				claim, err := tf.KubeClient.CoreV1().PersistentVolumeClaims(ns.Name).Get(volume.VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				claimCapacity := claim.Status.Capacity[v1.ResourceStorage]
				capacity := strings.TrimSuffix(claimCapacity.String(), "i")
				switch claim.Labels[selectors.LabelNamespaceKey] {
				case ns1.Name:
					Expect(capacity).To(Equal(ns1.Storage.Size))
				case ns2.Name:
					Expect(capacity).To(Equal(ns2.Storage.Size))
				}
			}
		}
	}

}

func testVolumeIsReused(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int) {
	Expect(nodeCount).To(BeNumerically(">", 1))
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pod, err := tf.KubeClient.CoreV1().Pods(ns.Name).Get(fmt.Sprintf("%s-%d", res.Name, nodeCount-1), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	var volumeName string
	for _, volume := range pod.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			claim, err := tf.KubeClient.CoreV1().PersistentVolumeClaims(ns.Name).Get(volume.VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			volumeName = claim.Spec.VolumeName
		}
	}
	Expect(volumeName).NotTo(BeEmpty())

	err = tf.ScaleCluster(res, nodeCount-1)
	Expect(err).NotTo(HaveOccurred())

	err = tf.ScaleCluster(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pod, err = tf.KubeClient.CoreV1().Pods(ns.Name).Get(fmt.Sprintf("%s-%d", res.Name, nodeCount-1), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	for _, volume := range pod.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil {
			claim, err := tf.KubeClient.CoreV1().PersistentVolumeClaims(ns.Name).Get(volume.VolumeSource.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(claim.Spec.VolumeName).To(Equal(volumeName))
		}
	}
}
