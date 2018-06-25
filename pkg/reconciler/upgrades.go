package reconciler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
)

func (r *AerospikeClusterReconciler) maybeUpgradePodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, index int) error {
	// check whether a pod with the specified index exists
	pod, err := r.getPodWithIndex(aerospikeCluster, index)
	if err != nil {
		// we've failed to get the pod with the specified index
		return err
	}
	if pod == nil {
		// no pod with the specified index exists, so we return
		return nil
	}
	// get the version of aerospike server running on the pod
	version, err := getAerospikeServerVersionFromPod(pod)
	if err != nil {
		return err
	}
	// skip the upgrade if the pod is already running the target version
	if version == aerospikeCluster.Spec.Version {
		return nil
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrading pod %s to version %s", meta.Key(pod), aerospikeCluster.Spec.Version)
	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeUpgradeStarted,
		"upgrading pod %s to version %s",
		meta.Key(pod), aerospikeCluster.Spec.Version)

	// restart the target pod
	newPod, err := r.safeRestartPodWithIndex(aerospikeCluster, configMap, index)
	if err != nil {
		return err
	}
	// ensure the pod has the target version
	version, err = getAerospikeServerVersionFromPod(newPod)
	if err != nil {
		return err
	}
	if version != aerospikeCluster.Spec.Version {
		r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeUpgradeFailed,
			"failed to upgrade pod %s to version %s",
			meta.Key(pod), aerospikeCluster.Spec.Version)
		return fmt.Errorf("failed to upgrade pod %s to version %s", meta.Key(newPod), aerospikeCluster.Spec.Version)
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgraded pod %s to version %s", meta.Key(pod), aerospikeCluster.Spec.Version)
	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeUpgradeFinished,
		"upgraded pod %s to version %s",
		meta.Key(pod), aerospikeCluster.Spec.Version)

	return nil
}

func (r *AerospikeClusterReconciler) signalBackupStarted(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionAutoBackupStarted,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupStarted,
		Message:            "cluster backup started",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusBackupAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonClusterAutoBackupStarted,
		"cluster backup started")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup started")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalBackupFinished(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionAutoBackupFinished,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupFinished,
		Message:            "cluster backup finished",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonClusterAutoBackupFinished,
		"cluster backup finished")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup finished")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalBackupFailed(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionAutoBackupFailed,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupFailed,
		Message:            "cluster backup failed",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonClusterAutoBackupFailed,
		"cluster backup failed")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup failed")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeStarted(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionUpgradeStarted,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeStarted,
		Message:            fmt.Sprintf("upgrade from version %s to %s started", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusStartedAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonClusterUpgradeStarted,
		"upgrade from version %s to %s started", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrade from version %s to %s started", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeFailed(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionUpgradeFailed,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeFailed,
		Message:            fmt.Sprintf("upgrade from version %s to %s failed", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusFailedAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonClusterUpgradeFailed,
		"upgrade from version %s to %s failed",
		aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrade from version %s to %s failed", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeFinished(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*aerospikev1alpha1.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               aerospikev1alpha1.ConditionUpgradeFinished,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeFinished,
		Message:            fmt.Sprintf("finished upgrade from version %s to %s", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	removeAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonClusterUpgradeFinished,
		"finished upgrade from version %s to %s", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("finished upgrade from version %s to %s", aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version)

	return aerospikeCluster, nil
}
