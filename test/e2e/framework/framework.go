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

package framework

import (
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
)

const (
	watchTimeout = 10 * time.Minute
)

type TestFramework struct {
	AerospikeClient *aerospikeclientset.Clientset
	KubeClient      *kubernetes.Clientset
}

func NewTestFramework(kubeconfigPath string) (*TestFramework, error) {
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
