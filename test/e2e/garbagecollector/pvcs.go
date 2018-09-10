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

	"github.com/travelaudience/aerospike-operator/pkg/asutils"
	astime "github.com/travelaudience/aerospike-operator/pkg/utils/time"
	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

func testDeleteExpiredPVC(tf *framework.TestFramework, ns *v1.Namespace, finalNodecount int32, ttl string) {
	aerospikeCluster := tf.NewAerospikeClusterWithDefaults()
	aerospikeCluster.Spec.NodeCount = finalNodecount + 1
	aerospikeCluster.Spec.Namespaces[0].Storage.PersistentVolumeClaimTTL = &ttl
	asc, err := tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(ns.Name).Create(&aerospikeCluster)
	Expect(err).NotTo(HaveOccurred())

	err = tf.WaitForClusterNodeCount(asc, aerospikeCluster.Spec.NodeCount)
	Expect(err).NotTo(HaveOccurred())

	lastPod, err := tf.KubeClient.CoreV1().Pods(asc.Namespace).Get(fmt.Sprintf("%s-%d", asc.Name, finalNodecount), metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	// get the PVC used by the last pod of the aerospikecluster
	// (the one will be deleted)
	var lastPVC *v1.PersistentVolumeClaim
	for _, pvc := range lastPod.Spec.Volumes {
		if pvc.PersistentVolumeClaim != nil {
			lastPVC, err = tf.KubeClient.CoreV1().PersistentVolumeClaims(lastPod.Namespace).Get(pvc.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
		}
	}

	err = tf.ScaleCluster(asc, finalNodecount)
	Expect(err).NotTo(HaveOccurred())

	asc, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeClusters(asc.Namespace).Get(asc.Name, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(asc.Status.NodeCount).To(Equal(finalNodecount))

	clusterSize, err := asutils.GetClusterSize(fmt.Sprintf("%s.%s", asc.Name, asc.Namespace), 3000)
	Expect(err).NotTo(HaveOccurred())
	Expect(int32(clusterSize)).To(Equal(finalNodecount))

	pvcDuration, err := astime.ParseDuration(ttl)
	Expect(err).NotTo(HaveOccurred())

	// wait for the backup to get expired.
	<-time.After(pvcDuration)

	// wait for the aerospikenamespacebackup to be deleted
	errCh := make(chan error, 1)
	go func(timeout time.Duration) {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := tf.KubeClient.CoreV1().PersistentVolumeClaims(lastPVC.Namespace).Get(lastPVC.Name, metav1.GetOptions{}); errors.IsNotFound(err) {
					errCh <- nil
					return
				}
			case <-time.After(timeout):
				errCh <- fmt.Errorf("timed out waiting for pvc to be deleted")
				return
			}
		}
	}(garbageCollectorTimeout)
	Expect(<-errCh).NotTo(HaveOccurred())
}
