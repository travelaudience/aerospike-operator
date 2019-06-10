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

	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/common"
)

type BackupRestoreObject interface {
	GetOperationType() common.OperationType
	GetKind() string
	GetName() string
	GetNamespace() string
	GetObjectMeta() *metav1.ObjectMeta
	GetStorage() *BackupStorageSpec
	SetStorage(*BackupStorageSpec)
	GetTarget() *TargetNamespace
	GetConditions() []apiextensions.CustomResourceDefinitionCondition
	SetConditions([]apiextensions.CustomResourceDefinitionCondition)
	GetFailedConditionType() apiextensions.CustomResourceDefinitionConditionType
	GetFinishedConditionType() apiextensions.CustomResourceDefinitionConditionType
	GetStartedConditionType() apiextensions.CustomResourceDefinitionConditionType
	SyncStatusWithSpec() bool
}
