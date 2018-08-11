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

package events

const (
	// ReasonValidationError is the reason used in corev1.Event objects that are related to
	// validation errors.
	ReasonValidationError = "ValidationError"

	// ReasonNodeStarting is the reason used in corev1.Event objects created when waiting
	// for pods to be running and ready.
	ReasonNodeStarting = "NodeStarting"

	// ReasonNodeStarted is the reason used in corev1.Event objects created when pods are
	// running and ready.
	ReasonNodeStarted = "NodeStarted"

	// ReasonNodeStartedFailed is the reason used in corev1.Event objects created when pods have
	// failed to start.
	ReasonNodeStartedFailed = "NodeStartedFailed"

	// ReasonNodeUpgradeStarted is the reason used in corev1.Event objects created when an
	// upgrade operation starts on a pod.
	ReasonNodeUpgradeStarted = "NodeUpgradeStarted"

	// ReasonNodeUpgradeFailed is the reason used in corev1.Event objects created when an
	// upgrade operation fails on a pod.
	ReasonNodeUpgradeFailed = "NodeUpgradeFailed"

	// ReasonNodeUpgradeFinished is the reason used in corev1.Event objects created when an
	// upgrade operation finishes on a pod.
	ReasonNodeUpgradeFinished = "NodeUpgradeFinished"

	// ReasonWaitForMigrationsStarted is the reason used in corev1.Event objects created when
	// migrations have started.
	ReasonWaitForMigrationsStarted = "WaitForMigrationsStarted"

	// ReasonWaitingForMigrations is the reason used in corev1.Event objects created when
	// waiting for migrations to finish.
	ReasonWaitingForMigrations = "WaitingForMigrations"

	// ReasonWaitForMigrationsFinished is the reason used in corev1.Event objects created when
	// migrations are finished.
	ReasonWaitForMigrationsFinished = "WaitForMigrationsFinished"

	// ReasonJobFinished is the reason used in corev1.Event objects indicating the backup or
	// restore is finished
	ReasonJobFinished = "JobFinished"

	// ReasonJobFailed is the reason used in corev1.Event objects indicating the backup or
	// restore has failed
	ReasonJobFailed = "JobFailed"

	// ReasonJobCreated is the reason used in corev1.Event objects indicating the backup or
	// restore job has been created
	ReasonJobCreated = "JobCreated"

	// ReasonClusterUpgradeStarted is the reason used in corev1.Event objects indicating that a
	// cluster upgrade has started
	ReasonClusterUpgradeStarted = "ClusterUpgradeStarted"

	// ReasonClusterUpgradeFailed is the reason used in corev1.Event objects indicating that a
	// cluster upgrade has failed
	ReasonClusterUpgradeFailed = "ClusterUpgradeFailed"

	// ReasonClusterUpgradeFinished is the reason used in corev1.Event objects indicating that a
	// cluster upgrade has finished
	ReasonClusterUpgradeFinished = "ClusterUpgradeFinished"

	// ReasonClusterAutoBackupStarted is the reason used in corev1.Event objects indicating that a
	// cluster backup has started
	ReasonClusterAutoBackupStarted = "ClusterAutoBackupStarted"

	// ReasonClusterAutoBackupFinished is the reason used in corev1.Event objects indicating that a
	// cluster backup has finished
	ReasonClusterAutoBackupFinished = "ClusterAutoBackupFinished"

	// ReasonClusterAutoBackupFailed is the reason used in corev1.Event objects indicating that a
	// cluster backup has failed
	ReasonClusterAutoBackupFailed = "ClusterAutoBackupFailed"
)
