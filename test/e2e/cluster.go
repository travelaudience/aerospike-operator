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

package e2e

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

var _ = Describe("AerospikeCluster", func() {
	var (
		ns *v1.Namespace
	)

	Context("in dedicated namespace", func() {
		BeforeEach(func() {
			var err error
			ns, err = tf.CreateRandomNamespace()
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := tf.DeleteNamespace(ns.Name)
			Expect(err).NotTo(HaveOccurred())
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
		It("cannot be created with len(spec.namespaces)==3", func() {
			testCreateAerospikeClusterWithThreeNamespaces(tf, ns)
		})
		It("cannot be created if spec.namespaces.replicationFactor[*] > spec.nodeCount", func() {
			testCreateAerospikeClusterWithInvalidReplicationFactor(tf, ns)
		})
		It("is created with the provided spec.nodeCount", func() {
			testCreateAerospikeClusterWithNodeCount(tf, ns, 1)
		})
		It("accepts connections on the service port", func() {
			testConnectToAerospikeCluster(tf, ns)
		})
		It("assigns each pod a persistent volume per Aerospike namespace with the requested size", func() {
			testVolumesSizeMatchNamespaceSpec(tf, ns, 1, 2, 4)
		})
		It("reuses the persistent volume of a deleted pod", func() {
			testVolumeIsReused(tf, ns, 2)
		})
	})
})

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

func testCreateAerospikeClusterWithNodeCount(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = nodeCount
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(res, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	pods, err := tf.KubeClient.CoreV1().Pods(ns.Name).List(listoptions.PodsByClusterName(res.Name))
	Expect(err).NotTo(HaveOccurred())
	Expect(len(pods.Items)).To(Equal(nodeCount))
}

func testConnectToAerospikeCluster(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	res, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	job, err := tf.KubeClient.BatchV1().Jobs(ns.Name).Create(&batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "asping",
			Namespace: ns.Name,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "aerospike-tools",
							Image:           "quay.io/travelaudience/aerospike-operator-tools:latest",
							ImagePullPolicy: v1.PullAlways,
							Command: []string{
								"asping",
								"-target-host", fmt.Sprintf("%s.%s.svc.cluster.local", res.Name, res.Namespace),
								"-target-port", "3000",
							},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
			BackoffLimit: pointers.NewInt32(10),
		},
	})
	Expect(err).NotTo(HaveOccurred())

	w, err := tf.KubeClient.BatchV1().Jobs(job.Namespace).Watch(listoptions.ObjectByName(job.Name))
	Expect(err).NotTo(HaveOccurred())
	last, err := watch.Until(2*time.Minute, w, func(event watch.Event) (bool, error) {
		return event.Object.(*batchv1.Job).Status.Succeeded == 1, nil
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(last).NotTo(BeNil())
}

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
	ns1 := tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1)
	aerospikeCluster.Spec.Namespaces = []v1alpha1.AerospikeNamespaceSpec{ns1}
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
