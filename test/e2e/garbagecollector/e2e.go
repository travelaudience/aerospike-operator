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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"time"

	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

const (
	// garbageCollectorTimeout specifies the timeout at which the tests
	// should stop checking if a resource was deleted by the garbage-collector
	garbageCollectorTimeout = time.Minute
)

var (
	tf *framework.TestFramework
)

func RegisterTestFramework(testFramework *framework.TestFramework) {
	tf = testFramework
}

var _ = Describe("GarbageCollector", func() {
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
			err := tf.DeleteNamespace(ns)
			Expect(err).NotTo(HaveOccurred())
		})

		It("deletes an expired aerospikenamespacebackup", func() {
			testDeleteExpiredAerospikeNamespaceBackup(tf, ns, "0.0001d")
		})

		It("does not delete an aerospikenamespacebackup with unspecified ttl", func() {
			testDoNotDeleteAerospikeNamespaceBackupWithoutTTL(tf, ns)
		})

		It("deletes an expired pvc", func() {
			testDeleteExpiredPVC(tf, ns, 1, "0.0001d")
		})
	})
})
