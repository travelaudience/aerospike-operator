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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/travelaudience/aerospike-operator/test/e2e/framework"
)

var (
	kubeconfig string
	tf         *framework.TestFramework
)

var _ = BeforeSuite(func() {
	var err error
	tf, err = framework.NewTestEnvironment(kubeconfig)
	Expect(err).NotTo(HaveOccurred())
	err = tf.SetUp()
	Expect(err).NotTo(HaveOccurred())
})

func RunE2ETests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "aerospike-operator e2e suite")
}

var _ = AfterSuite(func() {
	err := tf.TearDown()
	Expect(err).NotTo(HaveOccurred())
})
