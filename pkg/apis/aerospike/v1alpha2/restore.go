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
	"reflect"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// AerospikeNamespaceRestore represents a single restore operation targeting a single Aerospike namespace.
type AerospikeNamespaceRestore struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// The specification of the restore operation.
	Spec AerospikeNamespaceRestoreSpec `json:"spec"`
	// The status of the restore operation.
	Status AerospikeNamespaceRestoreStatus `json:"status"`
}

// AerospikeNamespaceRestoreSpec specifies the configuration for a restore operation.
type AerospikeNamespaceRestoreSpec struct {
	// The specification of the Aerospike cluster and namespace the backup will be restored to.
	Target TargetNamespace `json:"target"`
	// The specification of how the backup should be retrieved.
	// +optional
	Storage *BackupStorageSpec `json:"storage,omitempty"`
}

// AerospikeNamespaceRestoreStatus is the status for an AerospikeNamespaceRestore resource
type AerospikeNamespaceRestoreStatus struct {
	// The configuration for the restore operation.
	AerospikeNamespaceRestoreSpec
	// Details about the current condition of the AerospikeNamespaceRestore resource.
	// +k8s:openapi-gen=false
	Conditions []apiextensions.CustomResourceDefinitionCondition `json="conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceRestoreList is a list of AerospikeNamespaceRestore resources
type AerospikeNamespaceRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadaata.
	metav1.ListMeta `json:"metadata"`

	// The list of AerospikeNamespaceRestore resources.
	Items []AerospikeNamespaceRestore `json:"items"`
}

func (r *AerospikeNamespaceRestore) GetOperationType() common.OperationType {
	return common.OperationTypeRestore
}

func (r *AerospikeNamespaceRestore) GetKind() string {
	return common.AerospikeNamespaceRestoreKind
}

func (r *AerospikeNamespaceRestore) GetName() string {
	return r.Name
}

func (r *AerospikeNamespaceRestore) GetNamespace() string {
	return r.Namespace
}

func (r *AerospikeNamespaceRestore) GetObjectMeta() *metav1.ObjectMeta {
	return &r.ObjectMeta
}

func (r *AerospikeNamespaceRestore) GetStorage() *BackupStorageSpec {
	return r.Spec.Storage
}

func (r *AerospikeNamespaceRestore) SetStorage(storage *BackupStorageSpec) {
	r.Spec.Storage = storage
}

func (r *AerospikeNamespaceRestore) GetTarget() *TargetNamespace {
	return &r.Spec.Target
}

func (r *AerospikeNamespaceRestore) GetConditions() []apiextensions.CustomResourceDefinitionCondition {
	return r.Status.Conditions
}

func (r *AerospikeNamespaceRestore) SetConditions(newConditions []apiextensions.CustomResourceDefinitionCondition) {
	r.Status.Conditions = newConditions
}

func (b *AerospikeNamespaceRestore) GetFailedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionRestoreFailed
}

func (b *AerospikeNamespaceRestore) GetFinishedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionRestoreFinished
}

func (b *AerospikeNamespaceRestore) GetStartedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return common.ConditionRestoreStarted
}

func (b *AerospikeNamespaceRestore) SyncStatusWithSpec() bool {
	mustUpdate := false
	if !reflect.DeepEqual(b.Status.Storage, b.Spec.Storage) {
		b.Status.Storage = b.Spec.Storage
		mustUpdate = true
	}
	if !reflect.DeepEqual(b.Status.Target, b.Spec.Target) {
		b.Status.Target = b.Spec.Target
		mustUpdate = true
	}
	return mustUpdate
}
