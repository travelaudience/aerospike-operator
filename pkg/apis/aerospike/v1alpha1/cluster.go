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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeCluster is a specification for an AerospikeCluster resource
type AerospikeCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AerospikeClusterSpec   `json:"spec"`
	Status AerospikeClusterStatus `json:"status"`
}

// AerospikeClusterSpec is the spec for an AerospikeCluster resource
type AerospikeClusterSpec struct {
	NodeCount  int                      `json:"nodeCount"`
	Version    string                   `json:"version"`
	Namespaces []AerospikeNamespaceSpec `json:"namespaces"`
}

// AerospikeClusterStatus is the status for an AerospikeCluster resource
type AerospikeClusterStatus struct {
	AerospikeClusterSpec
}

// AerospikeNamespaceSpec is the spec for an AerospikeNamespace object
type AerospikeNamespaceSpec struct {
	Name              string      `json:"name"`
	ReplicationFactor int         `json:"replicationFactor"`
	MemorySize        string      `json:"memorySize"`
	DefaultTTL        string      `json:"defaultTTL"`
	Storage           StorageSpec `json:"storage"`
}

// StorageSpec is the spec for a Storage object
type StorageSpec struct {
	Type             string `json:"type"`
	Size             string `json:"size"`
	StorageClassName string `json:"storageClassName"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeClusterList is a list of AerospikeCluster resources
type AerospikeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AerospikeCluster `json:"items"`
}
