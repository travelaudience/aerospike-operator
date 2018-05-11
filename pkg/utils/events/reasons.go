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

	// ReasonMigrationsFinishing is the reason used in corev1.Event objects created when
	// waiting for migrations to finish.
	ReasonMigrationsFinishing = "MigrationsFinishing"

	// ReasonMigrationsFinished is the reason used in corev1.Event objects created when
	// migrations are finished.
	ReasonMigrationsFinished = "MigrationsFinished"

	// ReasonInvalidTarget is the reason used in corev1.Event objects indicating a target
	// cluster and namespace is not reachable or does not exist
	ReasonInvalidTarget = "InvalidTarget"

	// ReasonInvalidSecret is the reason used in corev1.Event objects indicating that the
	// secret is not valid
	ReasonInvalidSecret = "InvalidSecret"

	// ReasonJobCompleted is the reason used in corev1.Event objects indicating the backup or
	// restore is completed
	ReasonJobCompleted = "ReasonJobCompleted"

	// ReasonJobFailed is the reason used in corev1.Event objects indicating the backup or
	// restore has failed
	ReasonJobFailed = "ReasonJobFailed"

	// ReasonJobRunning is the reason used in corev1.Event objects indicating the backup or
	// restore is completed
	ReasonJobRunning = "ReasonJobRunning"
)
