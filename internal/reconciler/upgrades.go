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

package reconciler

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/logfields"
	"github.com/travelaudience/aerospike-operator/internal/meta"
	"github.com/travelaudience/aerospike-operator/internal/utils/events"
	"github.com/travelaudience/aerospike-operator/internal/versioning"
)

func (r *AerospikeClusterReconciler) maybeUpgradePodWithIndex(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, configMap *corev1.ConfigMap, index int, upgrade *versioning.VersionUpgrade) (*corev1.Pod, error) {
	// check whether a pod with the specified index exists
	pod, err := r.getPodWithIndex(aerospikeCluster, index)
	if err != nil {
		// we've failed to get the pod with the specified index
		return nil, err
	}
	if pod == nil {
		// no pod with the specified index exists, so we return
		return nil, nil
	}
	// get the version of aerospike server running on the pod
	version, err := getAerospikeServerVersionFromPod(pod)
	if err != nil {
		return nil, err
	}
	// skip the upgrade if the pod is already running the target version
	if version == aerospikeCluster.Spec.Version {
		return pod, nil
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrading pod %s to version %s", meta.Key(pod), aerospikeCluster.Spec.Version)
	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonNodeUpgradeStarted,
		"upgrading pod %s to version %s",
		meta.Key(pod), aerospikeCluster.Spec.Version)

	// restart the target pod
	newPod, err := r.safeRestartPodWithIndex(aerospikeCluster, configMap, index, upgrade)
	if err != nil {
		return nil, err
	}
	// ensure the pod has the target version
	version, err = getAerospikeServerVersionFromPod(newPod)
	if err != nil {
		return nil, err
	}
	if version != aerospikeCluster.Spec.Version {
		r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonNodeUpgradeFailed,
			"failed to upgrade pod %s to version %s",
			meta.Key(pod), aerospikeCluster.Spec.Version)
		return nil, fmt.Errorf("failed to upgrade pod %s to version %s", meta.Key(newPod), aerospikeCluster.Spec.Version)
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgraded pod %s to version %s", meta.Key(pod), aerospikeCluster.Spec.Version)
	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonNodeUpgradeFinished,
		"upgraded pod %s to version %s",
		meta.Key(pod), aerospikeCluster.Spec.Version)

	return newPod, nil
}

func (r *AerospikeClusterReconciler) signalBackupStarted(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionAutoBackupStarted,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupStarted,
		Message:            "cluster backup started",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAerospikeClusterAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusBackupAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonClusterAutoBackupStarted,
		"cluster backup started")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup started")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalBackupFinished(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionAutoBackupFinished,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupFinished,
		Message:            "cluster backup finished",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonClusterAutoBackupFinished,
		"cluster backup finished")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup finished")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalBackupFailed(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionAutoBackupFailed,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterAutoBackupFailed,
		Message:            "cluster backup failed",
		LastTransitionTime: metav1.NewTime(time.Now()),
	})

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonClusterAutoBackupFailed,
		"cluster backup failed")

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("cluster backup failed")

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeStarted(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, upgrade *versioning.VersionUpgrade) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionUpgradeStarted,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeStarted,
		Message:            fmt.Sprintf("upgrade from version %s to %s started", upgrade.Source, upgrade.Target),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAerospikeClusterAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusStartedAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonClusterUpgradeStarted,
		"upgrade from version %s to %s started", upgrade.Source, upgrade.Target)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrade from version %s to %s started", upgrade.Source, upgrade.Target)

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeFailed(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, upgrade *versioning.VersionUpgrade) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionUpgradeFailed,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeFailed,
		Message:            fmt.Sprintf("upgrade from version %s to %s failed", upgrade.Source, upgrade.Target),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	setAerospikeClusterAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey, UpgradeStatusFailedAnnotationValue)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeWarning, events.ReasonClusterUpgradeFailed,
		"upgrade from version %s to %s failed",
		upgrade.Source, upgrade.Target)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("upgrade from version %s to %s failed", upgrade.Source, upgrade.Target)

	return aerospikeCluster, nil
}

func (r *AerospikeClusterReconciler) signalUpgradeFinished(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, upgrade *versioning.VersionUpgrade) (*aerospikev1alpha2.AerospikeCluster, error) {
	// grab a copy of aerospikeCluster in its current state so we can later
	// create a patch
	oldCluster := aerospikeCluster.DeepCopy()

	appendCondition(aerospikeCluster, apiextensions.CustomResourceDefinitionCondition{
		Type:               common.ConditionUpgradeFinished,
		Status:             apiextensions.ConditionTrue,
		Reason:             events.ReasonClusterUpgradeFinished,
		Message:            fmt.Sprintf("finished upgrade from version %s to %s", upgrade.Source, upgrade.Target),
		LastTransitionTime: metav1.NewTime(time.Now()),
	})
	removeAerospikeClusterAnnotation(aerospikeCluster, UpgradeStatusAnnotationKey)

	if err := r.patchCluster(oldCluster, aerospikeCluster); err != nil {
		return nil, err
	}

	r.recorder.Eventf(aerospikeCluster, corev1.EventTypeNormal, events.ReasonClusterUpgradeFinished,
		"finished upgrade from version %s to %s", upgrade.Source, upgrade.Target)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("finished upgrade from version %s to %s", upgrade.Source, upgrade.Target)

	return aerospikeCluster, nil
}
