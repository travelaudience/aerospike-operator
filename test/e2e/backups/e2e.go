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

package backups

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

var _ = Describe("AerospikeNamespaceBackupRestore", func() {
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

		It("successfuly completes a backup and a restore of a namespace", func() {
			testNamespaceBackupRestore(tf, ns, 100000)
		})

		It("restores from a backup with a different namespace", func() {
			testNamespaceRestoreFromDifferentNamespace(tf, ns, 100000)
		})
	})
})
