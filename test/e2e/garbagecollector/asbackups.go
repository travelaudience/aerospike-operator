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

package garbagecollector

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	astime "github.com/travelaudience/aerospike-operator/pkg/utils/time"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testDeleteExpiredAerospikeNamespaceBackup(tf *framework.TestFramework, ns *v1.Namespace, ttl string) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asBackup := tf.NewAerospikeNamespaceBackupGCS(asc, asc.Spec.Namespaces[0].Name, pointers.NewString(ttl))
	backup, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(ns.Name).Create(&asBackup)
	Expect(err).NotTo(HaveOccurred())

	backupDuration, err := astime.ParseDuration(ttl)
	Expect(err).NotTo(HaveOccurred())

	// wait for the backup to get expired.
	<-time.After(backupDuration)

	// wait for the aerospikenamespacebackup to be deleted
	errCh := make(chan error, 1)
	go func(timeout time.Duration) {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
					errCh <- nil
					return
				}
			case <-timer.C:
				errCh <- fmt.Errorf("timed out waiting for aerospikenamespacebackup to be deleted")
				return
			}
		}
	}(garbageCollectorTimeout)
	Expect(<-errCh).NotTo(HaveOccurred())
}

func testDoNotDeleteAerospikeNamespaceBackupWithoutTTL(tf *framework.TestFramework, ns *v1.Namespace) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	asBackup := tf.NewAerospikeNamespaceBackupGCS(asc, asc.Spec.Namespaces[0].Name, nil)
	backup, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(ns.Name).Create(&asBackup)
	Expect(err).NotTo(HaveOccurred())

	// wait to make sure the aerospikenamespacebackup is not deleted
	errCh := make(chan error, 1)
	go func(timeout time.Duration) {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(backup.Namespace).Get(backup.Name, metav1.GetOptions{}); err != nil {
					errCh <- err
					return
				}
			case <-timer.C:
				errCh <- nil
				return
			}
		}
	}(garbageCollectorTimeout)
	// if an error did not occur the resource still exists
	Expect(<-errCh).NotTo(HaveOccurred())
}
