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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
)

type TestFramework struct {
	AerospikeClient *aerospikeclientset.Clientset
	KubeClient      *kubernetes.Clientset

	podName string
}

func NewTestEnvironment(kubeconfigPath string) (*TestFramework, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	aerospikeClient, err := aerospikeclientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &TestFramework{
		AerospikeClient: aerospikeClient,
		KubeClient:      kubeClient,
	}, nil
}

func (tf *TestFramework) SetUp() error {
	return tf.createOperator()
}

func (tf *TestFramework) TearDown() error {
	return tf.deleteOperator()
}
