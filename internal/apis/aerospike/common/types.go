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

package common

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
)

const (
	// StorageTypeFile defines the file storage type for a given Aerospike namespace.
	StorageTypeFile = "file"

	// StorageTypeDevice defines the device storage type for a given Aerospike namespace.
	StorageTypeDevice = "device"

	// StorageTypeGCS defines the Google Cloud Storage type for a given Aerospike backup.
	StorageTypeGCS = "gcs"

	// ConditionBackupFailed defines a status condition that indicates that a backup job has failed
	ConditionBackupFailed apiextensions.CustomResourceDefinitionConditionType = "BackupFailed"

	// ConditionBackupFinished defines a status condition that indicates that a backup job has finished
	ConditionBackupFinished apiextensions.CustomResourceDefinitionConditionType = "BackupFinished"

	// ConditionBackupStarted defines a status condition that indicates that a backup job has started
	ConditionBackupStarted apiextensions.CustomResourceDefinitionConditionType = "BackupStarted"

	// ConditionRestoreFailed defines a status condition that indicates that a restore job has failed
	ConditionRestoreFailed apiextensions.CustomResourceDefinitionConditionType = "RestoreFailed"

	// ConditionRestoreFinished defines a status condition that indicates that a restore job has finished
	ConditionRestoreFinished apiextensions.CustomResourceDefinitionConditionType = "RestoreFinished"

	// ConditionRestoreStarted defines a status condition that indicates that a restore job has started
	ConditionRestoreStarted apiextensions.CustomResourceDefinitionConditionType = "RestoreStarted"

	// ConditionUpgradeStarted defines a status condition that indicates that an upgrade to an
	// Aerospike cluster has started
	ConditionUpgradeStarted apiextensions.CustomResourceDefinitionConditionType = "UpgradeStarted"

	// ConditionUpgradeFinished defines a status condition that indicates that an upgrade to an
	// Aerospike cluster has finished
	ConditionUpgradeFinished apiextensions.CustomResourceDefinitionConditionType = "UpgradeFinished"

	// ConditionUpgradeFailed defines a status condition that indicates that an upgrade to an
	// Aerospike cluster has failed
	ConditionUpgradeFailed apiextensions.CustomResourceDefinitionConditionType = "UpgradeFailed"

	// ConditionAutoBackupStarted defines a status condition that indicates that a pre-upgrade
	// backup for an Aerospike cluster has started
	ConditionAutoBackupStarted apiextensions.CustomResourceDefinitionConditionType = "AutoBackupStarted"

	// ConditionAutoBackupFinished defines a status condition that indicates that a pre-upgrade
	// backup for an Aerospike cluster has finished
	ConditionAutoBackupFinished apiextensions.CustomResourceDefinitionConditionType = "AutoBackupFinished"

	// ConditionAutoBackupFailed defines a status condition that indicates that a pre-upgrade
	// backup for an Aerospike cluster has failed
	ConditionAutoBackupFailed apiextensions.CustomResourceDefinitionConditionType = "AutoBackupFailed"

	// DefaultSecretFilename represents the name of the file that is required to exist
	// in the secret referenced in BackupStorageSpec objects.
	DefaultSecretFilename = "key.json"
)

// OperationType represents the type used to indicate whether a
// BackupRestoreObject represents a backup or a restore operation
type OperationType string

const (
	OperationTypeBackup  OperationType = "backup"
	OperationTypeRestore OperationType = "restore"

	AerospikeClusterKind          = "AerospikeCluster"
	AerospikeNamespaceBackupKind  = "AerospikeNamespaceBackup"
	AerospikeNamespaceRestoreKind = "AerospikeNamespaceRestore"
)
