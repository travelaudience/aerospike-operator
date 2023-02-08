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

package backups

import (
	"context"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func testNamespaceBackupRestore(tf *framework.TestFramework, ns *v1.Namespace, nRecords int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	asBackup := tf.NewAerospikeNamespaceBackupGCS(asc, asc.Spec.Namespaces[0].Name, nil)
	backup, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(ns.Name).Create(context.TODO(), &asBackup, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(backup)
	Expect(err).NotTo(HaveOccurred())

	// NewAerospikeClusterWithDefaults uses generated names. Hence, the restore is always made to a different cluster.
	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asRestore := tf.NewAerospikeNamespaceRestoreGCS(asc, asc.Spec.Namespaces[0].Name, backup)
	restore, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(ns.Name).Create(context.TODO(), &asRestore, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(restore)
	Expect(err).NotTo(HaveOccurred())

	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()
}

func testNamespaceRestoreFromDifferentNamespace(tf *framework.TestFramework, ns *v1.Namespace, nRecords int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.Namespaces[0] = tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-0", 1, 1, 0, 1)
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	asBackup := tf.NewAerospikeNamespaceBackupGCS(asc, asc.Spec.Namespaces[0].Name, nil)
	backup, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(ns.Name).Create(context.TODO(), &asBackup, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(backup)
	Expect(err).NotTo(HaveOccurred())

	// use a different name for the target namespace
	aerospikeCluster.Spec.Namespaces[0] = tf.NewAerospikeNamespaceWithFileStorage("aerospike-namespace-1", 1, 1, 0, 1)

	// NewAerospikeClusterWithDefaults uses generated names. Hence, the restore is always made to a different cluster.
	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asRestore := tf.NewAerospikeNamespaceRestoreGCS(asc, asc.Spec.Namespaces[0].Name, backup)
	restore, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(ns.Name).Create(context.TODO(), &asRestore, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(restore)
	Expect(err).NotTo(HaveOccurred())

	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()
}

func testNamespaceBackupRestoreWithoutBackupStorageSpec(tf *framework.TestFramework, ns *v1.Namespace, nRecords int) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.BackupSpec = &v1alpha2.AerospikeClusterBackupSpec{
		Storage: v1alpha2.BackupStorageSpec{
			Type:            common.StorageTypeGCS,
			Bucket:          framework.GCSBucketName,
			Secret:          framework.GCSSecretName,
			SecretNamespace: &framework.GCSSecretNamespace,
			SecretKey:       &framework.GCSSecretKey,
		},
	}
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	c1, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c1.WriteSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c1.Close()

	asBackup := tf.NewAerospikeNamespaceBackupGCSWithoutBackupStorageSpec(asc, asc.Spec.Namespaces[0].Name, nil)
	backup, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(ns.Name).Create(context.TODO(), &asBackup, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(backup)
	Expect(err).NotTo(HaveOccurred())

	// NewAerospikeClusterWithDefaults uses generated names. Hence, the restore is always made to a different cluster.
	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(context.TODO(), &aerospikeCluster, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asRestore := tf.NewAerospikeNamespaceRestoreGCSWithoutBackupStorageSpec(asc, asc.Spec.Namespaces[0].Name, backup)
	restore, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(ns.Name).Create(context.TODO(), &asRestore, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForBackupRestoreCompleted(restore)
	Expect(err).NotTo(HaveOccurred())

	c2, err := framework.NewAerospikeClient(asc)
	Expect(err).NotTo(HaveOccurred())
	err = c2.ReadSequentialIntegers(asc.Spec.Namespaces[0].Name, nRecords)
	Expect(err).NotTo(HaveOccurred())
	c2.Close()
}
