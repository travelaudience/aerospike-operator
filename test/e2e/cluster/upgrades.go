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
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/reconciler"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testNoDataLossOnAerospikeUpgrade(tf *framework.TestFramework, ns *v1.Namespace, nodeCount int32, nRecords int, sourceVersion, targetVersion string) {
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

	// write data to the first namespace of the Aerospike cluster
	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(aerospikeCluster.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	// get the latest resource version of the Aerospike cluster
	asc, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc.Namespace).Get(asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	// upgrade Aerospike cluster to targetVersion
	asc, err = tf.UpgradeClusterAndWait(asc, targetVersion)
	Expect(err).NotTo(HaveOccurred())

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
