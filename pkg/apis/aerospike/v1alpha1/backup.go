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
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// AerospikeNamespaceBackup represents a single backup operation targeting a single Aerospike namespace.
type AerospikeNamespaceBackup struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The specification of the backup operation.
	Spec AerospikeNamespaceBackupSpec `json:"spec"`
	// The status of the backup operation.
	Status AerospikeNamespaceBackupStatus `json:"status"`
}

// AerospikeNamespaceBackupSpec specifies the configuration for a backup operation.
type AerospikeNamespaceBackupSpec struct {
	// The specification of the Aerospike cluster and Aerospike namespace to backup.
	Target TargetNamespace `json:"target"`
	// The specification of how the backup will be stored.
	// +optional
	Storage *BackupStorageSpec `json:"storage,omitempty"`
	// The retention period (days) during which to keep backup data in cloud storage, suffixed with d.
	// Defaults to 0d, meaning the backup data will be kept forever.
	// +optional
	TTL *string `json:"ttl,omitempty"`
}

// TargetNamespace specifies the Aerospike cluster and namespace a single backup or restore operation will target.
type TargetNamespace struct {
	// The name of the Aerospike cluster against which the backup/restore operation will be performed.
	Cluster string `json:"cluster"`
	// The name of the Aerospike namespace to backup/restore.
	Namespace string `json:"namespace"`
}

// BackupStorageSpec specifies the configuration for the storage of a backup.
type BackupStorageSpec struct {
	// The type of cloud storage to use for the backup (e.g., gcs).
	Type string `json:"type"`
	// The name of the bucket where the backup is stored.
	Bucket string `json:"bucket"`
	// The name of the secret containing credentials to access the bucket.
	Secret string `json:"secret"`
	// The namespace to which the secret containing the credentials belongs to.
	// +optional
	SecretNamespace *string `json:"secretNamespace,omitempty"`
	// The name of the file in which the credentials are stored.
	// +optional
	SecretKey *string `json:"secretKey,omitempty"`
}

func (b *BackupStorageSpec) GetSecret() string {
	return b.Secret
}

func (b *BackupStorageSpec) GetSecretKey() string {
	if b.SecretKey != nil {
		return *b.SecretKey
	}
	return common.DefaultSecretFilename
}

func (b *BackupStorageSpec) GetSecretNamespace(fallbackNamespace string) string {
	namespace := fallbackNamespace
	if b.SecretNamespace != nil {
		namespace = *b.SecretNamespace
	}
	if namespace == "" {
		return metav1.NamespaceDefault
	}
	return namespace
}

// AerospikeNamespaceBackupStatus is the status for an AerospikeNamespaceBackup resource.
type AerospikeNamespaceBackupStatus struct {
	// The configuration for the backup operation.
	AerospikeNamespaceBackupSpec
	// Details about the current condition of the AerospikeNamespaceBackup resource.
	// +k8s:openapi-gen=false
	Conditions []apiextensions.CustomResourceDefinitionCondition `json="conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceBackupList represents a list of AerospikeNamespaceBackup resources.
type AerospikeNamespaceBackupList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	metav1.ListMeta `json:"metadata"`

	// The list of AerospikeNamespaceBackup resources.
	Items []AerospikeNamespaceBackup `json:"items"`
}

func (b *AerospikeNamespaceBackup) GetOperationType() common.OperationType {
	return common.OperationTypeBackup
}

func (b *AerospikeNamespaceBackup) GetKind() string {
	return common.AerospikeNamespaceBackupKind
}

func (b *AerospikeNamespaceBackup) GetName() string {
	return b.Name
}

func (b *AerospikeNamespaceBackup) GetNamespace() string {
	return b.Namespace
}

func (b *AerospikeNamespaceBackup) GetObjectMeta() *metav1.ObjectMeta {
	return &b.ObjectMeta
}

func (b *AerospikeNamespaceBackup) GetStorage() *BackupStorageSpec {
	return b.Spec.Storage
}

func (b *AerospikeNamespaceBackup) SetStorage(storage *BackupStorageSpec) {
	b.Spec.Storage = storage
}

func (b *AerospikeNamespaceBackup) GetTarget() *TargetNamespace {
	return &b.Spec.Target
}

func (b *AerospikeNamespaceBackup) GetConditions() []apiextensions.CustomResourceDefinitionCondition {
	return b.Status.Conditions
}

func (b *AerospikeNamespaceBackup) SetConditions(newConditions []apiextensions.CustomResourceDefinitionCondition) {
	b.Status.Conditions = newConditions
}

func (b *AerospikeNamespaceBackup) GetFailedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionBackupFailed
}

func (b *AerospikeNamespaceBackup) GetFinishedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionBackupFinished
}

func (b *AerospikeNamespaceBackup) GetStartedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionBackupStarted
}
