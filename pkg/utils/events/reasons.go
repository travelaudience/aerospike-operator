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
)
