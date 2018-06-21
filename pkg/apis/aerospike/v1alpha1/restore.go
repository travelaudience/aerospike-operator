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
	Target TargetNamespace `json:"target"`
	// +optional
	Storage *BackupStorageSpec `json:"storage,omitempty"`
}

// AerospikeNamespaceRestoreStatus is the status for an AerospikeNamespaceRestore resource
type AerospikeNamespaceRestoreStatus struct {
	AerospikeNamespaceRestoreSpec
	Conditions []apiextensions.CustomResourceDefinitionCondition `json="conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AerospikeNamespaceRestoreList is a list of AerospikeNamespaceRestore resources
type AerospikeNamespaceRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []AerospikeNamespaceRestore `json:"items"`
}

func (r *AerospikeNamespaceRestore) GetAction() ActionType {
	return ActionTypeRestore
}

func (r *AerospikeNamespaceRestore) GetKind() string {
	return AerospikeNamespaceRestoreKind
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
	return ConditionRestoreFailed
}

func (b *AerospikeNamespaceRestore) GetFinishedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return ConditionRestoreFinished
}

func (b *AerospikeNamespaceRestore) GetStartedConditionType() apiextensions.CustomResourceDefinitionConditionType {
	return ConditionRestoreStarted
}
