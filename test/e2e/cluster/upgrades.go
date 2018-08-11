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

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/asutils"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/reconciler"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testReusePVCsAndNoDataLossOnAerospikeUpgrade(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32, nRecords int, sourceVersion, targetVersion string) {
	// create an Aerospike cluster with required parameters
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Version = sourceVersion
	aerospikeCluster.Spec.BackupSpec = &v1alpha1.AerospikeClusterBackupSpec{
		Storage: v1alpha1.BackupStorageSpec{
			Type:   v1alpha1.StorageTypeGCS,
			Bucket: framework.GCSBucketName,
			Secret: framework.GCSSecretName,
		},
	}
	aerospikeCluster.Spec.NodeCount = nodeCount
	aerospikeCluster.Spec.Namespaces[0].ReplicationFactor = pointers.NewInt32(2)
	asc, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	// wait until the Aerospiek cluster is ready
	err = tf.WaitForClusterNodeCount(asc, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	// get the current list of pods belonging to the Aerospike cluster
	preUpgradePods, err := tf.KubeClient.CoreV1().Pods(aerospikeCluster.Namespace).List(listoptions.ResourcesByClusterName(aerospikeCluster.Name))
	Expect(err).NotTo(HaveOccurred())

	// write data to the first namespace of the Aerospike cluster
	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	// get the latest version of the aerospikecluster resource
	asc, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc.Namespace).Get(asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	// upgrade Aerospike cluster to targetVersion
	asc, err = tf.UpgradeClusterAndWait(asc, targetVersion)
	Expect(err).NotTo(HaveOccurred())

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(asc.Status.NodeCount))

	// get the current list of pods belonging to the Aerospike cluster
	postUpgradePods, err := tf.KubeClient.CoreV1().Pods(aerospikeCluster.Namespace).List(listoptions.ResourcesByClusterName(aerospikeCluster.Name))
	Expect(err).NotTo(HaveOccurred())

	// ensure that each pod re-uses the existing persistent volume claim
	for _, pod := range preUpgradePods.Items {
		for _, newPod := range postUpgradePods.Items {
			if pod.Name == newPod.Name {
				Expect(len(pod.Spec.Volumes)).To(Equal(len(aerospikeCluster.Spec.Namespaces)))
				Expect(len(newPod.Spec.Volumes)).To(Equal(len(aerospikeCluster.Spec.Namespaces)))
				for _, vol := range pod.Spec.Volumes {
					for _, newVol := range newPod.Spec.Volumes {
						if vol.Name == newVol.Name {
							Expect(vol.PersistentVolumeClaim.ClaimName).To(Equal(newVol.PersistentVolumeClaim.ClaimName))
						}
					}
				}
			}
		}
	}

	// read data from the first namespace of the Aerospike cluster
	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()

	// check if an AerospikeNamespaceBackup exists for each of the namespaces of the Aerospike cluster
	for _, namespace := range asc.Spec.Namespaces {
		backup, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeNamespaceBackups(ns.Name).Get(reconciler.GetBackupName(namespace.Name, sourceVersion, targetVersion), metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		completed := false
		for _, condition := range backup.Status.Conditions {
			if condition.Type == v1alpha1.ConditionBackupFinished &&
				condition.Status == apiextensions.ConditionTrue {
				completed = true
			}
		}
		Expect(completed).To(Equal(true))
	}
}

func testRecreatePVCsAndNoDataLossOnAerospikeUpgrade(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32, nRecords int, sourceVersion, targetVersion string) {
	// create an Aerospike cluster with required parameters
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Version = sourceVersion
	aerospikeCluster.Spec.BackupSpec = &v1alpha1.AerospikeClusterBackupSpec{
		Storage: v1alpha1.BackupStorageSpec{
			Type:   v1alpha1.StorageTypeGCS,
			Bucket: framework.GCSBucketName,
			Secret: framework.GCSSecretName,
		},
	}
	aerospikeCluster.Spec.NodeCount = nodeCount
	aerospikeCluster.Spec.Namespaces[0].ReplicationFactor = pointers.NewInt32(2)
	asc, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	// wait until the Aerospiek cluster is ready
	err = tf.WaitForClusterNodeCount(asc, nodeCount)
	Expect(err).NotTo(HaveOccurred())

	// get the current list of pods belonging to the Aerospike cluster
	preUpgradePods, err := tf.KubeClient.CoreV1().Pods(aerospikeCluster.Namespace).List(listoptions.ResourcesByClusterName(aerospikeCluster.Name))
	Expect(err).NotTo(HaveOccurred())

	// write data to the first namespace of the Aerospike cluster
	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	// get the latest version of the aerospikecluster resource
	asc, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc.Namespace).Get(asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	// upgrade Aerospike cluster to targetVersion
	asc, err = tf.UpgradeClusterAndWait(asc, targetVersion)
	Expect(err).NotTo(HaveOccurred())

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(asc.Status.NodeCount))

	// get the current list of pods belonging to the Aerospike cluster
	postUpgradePods, err := tf.KubeClient.CoreV1().Pods(aerospikeCluster.Namespace).List(listoptions.ResourcesByClusterName(aerospikeCluster.Name))
	Expect(err).NotTo(HaveOccurred())

	// ensure that each pod uses a new persistent volume claim
	for _, pod := range preUpgradePods.Items {
		for _, newPod := range postUpgradePods.Items {
			if pod.Name == newPod.Name {
				Expect(len(pod.Spec.Volumes)).To(Equal(len(aerospikeCluster.Spec.Namespaces)))
				Expect(len(newPod.Spec.Volumes)).To(Equal(len(aerospikeCluster.Spec.Namespaces)))
				for _, vol := range pod.Spec.Volumes {
					for _, newVol := range newPod.Spec.Volumes {
						if vol.Name == newVol.Name {
							Expect(vol.PersistentVolumeClaim.ClaimName).NotTo(Equal(newVol.PersistentVolumeClaim.ClaimName))
						}
					}
				}
			}
		}
	}

	// read data from the first namespace of the Aerospike cluster
	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()

	// check if an AerospikeNamespaceBackup exists for each of the namespaces of the Aerospike cluster
	for _, namespace := range asc.Spec.Namespaces {
		backup, err := tf.AerospikeClient.AerospikeV1alpha1().AerospikeNamespaceBackups(ns.Name).Get(reconciler.GetBackupName(namespace.Name, sourceVersion, targetVersion), metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		completed := false
		for _, condition := range backup.Status.Conditions {
			if condition.Type == v1alpha1.ConditionBackupFinished &&
				condition.Status == apiextensions.ConditionTrue {
				completed = true
			}
		}
		Expect(completed).To(Equal(true))
	}
}
