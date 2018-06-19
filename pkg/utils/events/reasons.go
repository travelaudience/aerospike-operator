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

	// ReasonInvalidTarget is the reason used in corev1.Event objects indicating a target
	// cluster and namespace is not reachable or does not exist
	ReasonInvalidTarget = "InvalidTarget"

	// ReasonInvalidSecret is the reason used in corev1.Event objects indicating that the
	// secret is not valid
	ReasonInvalidSecret = "InvalidSecret"

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
)
