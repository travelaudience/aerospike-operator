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
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	//StorageTypeFile defines the file storage type for a given Aerospike namespace.
	StorageTypeFile = "file"

	//StorageTypeDevice defines the device storage type for a given Aerospike namespace.
	StorageTypeDevice = "device"

	//StorageTypeGCS defines the Google Cloud Storage type for a given Aerospike backup.
	StorageTypeGCS = "gcs"

	//ConditionCompleted defines a status condition to indicate when a backup or restore
	//job has been completed
	ConditionCompleted apiextensions.CustomResourceDefinitionConditionType = "Completed"

	//ConditionCreated defines a status condition to indicate when a backup or restore job
	//has been created
	ConditionCreated apiextensions.CustomResourceDefinitionConditionType = "Created"

	//ConditionExpired defines a status condition to indicate when a backup or restore job
	//has been expired
	ConditionExpired apiextensions.CustomResourceDefinitionConditionType = "Expired"
)

type ActionType string

const (
	ActionTypeBackup  ActionType = "backup"
	ActionTypeRestore ActionType = "restore"
)

type BackupRestoreObject interface {
	GetAction() ActionType
	GetType() string
	GetName() string
	GetNamespace() string
	GetObjectMeta() *v1.ObjectMeta
	GetStorage() *BackupStorageSpec
	GetTarget() *TargetNamespace
	GetConditions() []apiextensions.CustomResourceDefinitionCondition
}
