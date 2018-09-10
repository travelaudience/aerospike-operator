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

package v1alpha2

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// AerospikeCluster represents an Aerospike cluster.
type AerospikeCluster struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The specification of the Aerospike cluster.
	Spec AerospikeClusterSpec `json:"spec"`
	// The status of the Aerospike cluster.
	Status AerospikeClusterStatus `json:"status"`
}

// AerospikeClusterSpec specifies the desired state of an Aerospike cluster.
type AerospikeClusterSpec struct {
	// The number of nodes in the Aerospike cluster.
	NodeCount int32 `json:"nodeCount"`
	// The version of Aerospike to be deployed.
	Version string `json:"version"`
	// The specification of the Aerospike namespaces in the cluster.
	// Must have exactly one element.
	Namespaces []AerospikeNamespaceSpec `json:"namespaces"`
	// The specification of how Aerospike namespace backups made by aerospike-operator should be performed and stored.
	// It is only required to be present if one wants to perform version upgrades on the Aerospike cluster.
	// +optional
	BackupSpec *AerospikeClusterBackupSpec `json:"backupSpec,omitempty"`
}

// AerospikeClusterStatus represents the current state of an Aerospike cluster.
type AerospikeClusterStatus struct {
	// The desired state of the Aerospike cluster.
	AerospikeClusterSpec
	// Details about the current condition of the AerospikeCluster resource.
	// +k8s:openapi-gen=false
	Conditions []apiextensions.CustomResourceDefinitionCondition `json:"conditions"`
}

// AerospikeNamespaceSpec specifies the configuration for an Aerospike namespace.
type AerospikeNamespaceSpec struct {
	// The name of the Aerospike namespace.
	Name string `json:"name"`
	// The number of replicas (including the master copy) for this Aerospike namespace.
	// If absent, the default value provided by Aerospike will be used.
	// +optional
	ReplicationFactor *int32 `json:"replicationFactor,omitempty"`
	// The amount of memory (gibibytes) to be used for index and data, suffixed with G.
	// If absent, the default value provided by Aerospike will be used.
	// +optional
	MemorySize *string `json:"memorySize,omitempty"`
	// Default record time-to-live (seconds) since it is created or last updated, suffixed with s.
	// When TTL is reached, the record is deleted automatically.
	// A TTL of 0s means the record never expires.
	// If absent, the default value provided by Aerospike will be used.
	// +optional
	DefaultTTL *string `json:"defaultTTL,omitempty"`
	// Specifies how data for the Aerospike namespace will be stored.
	Storage StorageSpec `json:"storage"`
}

// AerospikeClusterBackupSpec specifies how Aerospike namespace backups made by aerospike-operator before a version upgrade should be stored.
type AerospikeClusterBackupSpec struct {
	// The retention period (days) during which to keep backup data in cloud storage, suffixed with d.
	// Defaults to 0d, meaning the backup data will be kept forever.
	// +optional
	TTL *string `json:"ttl,omitempty"`
	// Specifies how the backup should be stored.
	Storage BackupStorageSpec `json:"storage"`
}

// StorageSpec specifies how data in a given Aerospike namespace will be stored.
type StorageSpec struct {
	// The storage engine to be used for the namespace (file or device).
	Type string `json:"type"`
	// The size (gibibytes) of the persistent volume to use for storing data in this namespace, suffixed with G.
	Size string `json:"size"`
	// The name of the storage class to use to create persistent volumes.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
	// The retention period (days) during which to keep PVCs for being
	// re-used after unmounted. Defaults to 0d, meaning the PVCs will be
	// kept forever.
	// +optional
	PersistentVolumeClaimTTL *string `json:"persistentVolumeClaimTTL,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeClusterList represents a list of Aerospike clusters.
type AerospikeClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	metav1.ListMeta `json:"metadata"`

	// The list of AerospikeCluster resources.
	Items []AerospikeCluster `json:"items"`
}
