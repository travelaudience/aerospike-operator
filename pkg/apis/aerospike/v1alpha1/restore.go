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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceRestore is a specification for an AerospikeNamespaceRestore resource
type AerospikeNamespaceRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AerospikeNamespaceRestoreSpec   `json:"spec"`
	Status AerospikeNamespaceRestoreStatus `json:"status"`
}

// AerospikeNamespaceRestoreSpec is the spec for an AerospikeNamespaceRestore resource
type AerospikeNamespaceRestoreSpec struct {
}

// AerospikeNamespaceRestoreStatus is the status for an AerospikeNamespaceRestore resource
type AerospikeNamespaceRestoreStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceRestoreList is a list of AerospikeNamespaceRestore resources
type AerospikeNamespaceRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AerospikeNamespaceRestore `json:"items"`
}
