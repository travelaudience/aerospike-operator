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

// AerospikeNamespaceBackup is a specification for an AerospikeNamespaceBackup resource
type AerospikeNamespaceBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AerospikeNamespaceBackupSpec   `json:"spec"`
	Status AerospikeNamespaceBackupStatus `json:"status"`
}

// AerospikeNamespaceBackupSpec is the spec for an AerospikeNamespaceBackup resource
type AerospikeNamespaceBackupSpec struct {
	Target  TargetNamespace   `json:"target"`
	Storage BackupStorageSpec `json:"storage"`
	TTL     string            `json:"ttl"`
}

// TargetNamespace specifies the cluster and namespace to backup
type TargetNamespace struct {
	Cluster   string `json:"cluster"`
	Namespace string `json:"namespace"`
}

// BackupStorageSpec specifies how the backup will be stored
type BackupStorageSpec struct {
	Type   string `json:"type"`
	Bucket string `json:"bucket"`
	Secret string `json:"secret"`
}

// AerospikeNamespaceBackupStatus is the status for an AerospikeNamespaceBackup resource
type AerospikeNamespaceBackupStatus struct {
	AerospikeNamespaceBackupSpec
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceBackupList is a list of AerospikeNamespaceBackup resources
type AerospikeNamespaceBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AerospikeNamespaceBackup `json:"items"`
}
