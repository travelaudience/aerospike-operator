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

package framework

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	randomNamespacePrefix = "aerospike-operator-e2e-"
)

func (tf *TestFramework) CreateRandomNamespace() (*v1.Namespace, error) {
	return tf.KubeClient.CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: randomNamespacePrefix,
		},
	})
}

func (tf *TestFramework) DeleteNamespace(name string) error {
	return tf.KubeClient.CoreV1().Namespaces().Delete(name, metav1.NewDeleteOptions(0))
}
