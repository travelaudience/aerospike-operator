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

package backuprestore

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/debug"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	"github.com/travelaudience/aerospike-operator/pkg/version"
)

const (
	// jobBackoffLimit is the maximum number of failures we tolerate before the
	// backup/restore job fails permanently.
	jobBackoffLimit = 3
)

// createJob creates the job associated with obj.
func (h *AerospikeBackupRestoreHandler) createJob(obj aerospikev1alpha1.BackupRestoreObject) (*batchv1.Job, error) {
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: h.getJobName(obj),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelClusterKey:   obj.GetTarget().Cluster,
				selectors.LabelNamespaceKey: obj.GetTarget().Namespace,
			},
			Namespace: obj.GetObjectMeta().Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.SchemeGroupVersion.String(),
					Kind:               obj.GetKind(),
					Name:               obj.GetObjectMeta().Name,
					UID:                obj.GetObjectMeta().UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(obj.GetAction()),
					Namespace: obj.GetObjectMeta().Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "aerospike-operator-tools",
							Image:           fmt.Sprintf("%s:%s", "quay.io/travelaudience/aerospike-operator-tools", version.Version),
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"backup",
								string(obj.GetAction()),
								fmt.Sprintf("-debug=%t", debug.DebugEnabled),
								fmt.Sprintf("-bucket-name=%s", obj.GetStorage().Bucket),
								fmt.Sprintf("-name=%s", obj.GetObjectMeta().Name),
								fmt.Sprintf("-secret-path=%s/%s", secretVolumeMountPath, secretFilename),
								fmt.Sprintf("-host=%s.%s", obj.GetTarget().Cluster, obj.GetNamespace()),
								fmt.Sprintf("-namespace=%s", obj.GetTarget().Namespace),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      secretVolumeName,
									ReadOnly:  true,
									MountPath: secretVolumeMountPath,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: secretVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: obj.GetStorage().Secret,
								},
							},
						},
					},
				},
			},
			BackoffLimit: pointers.NewInt32(jobBackoffLimit),
		},
	}

	res, err := h.kubeclientset.BatchV1().Jobs(obj.GetObjectMeta().Namespace).Create(&job)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		logfields.Job: meta.Key(res),
	}).Debugf("%s job created", obj.GetAction())
	return res, nil
}

// getJobName returns the name of the job associated with obj.
func (h *AerospikeBackupRestoreHandler) getJobName(obj aerospikev1alpha1.BackupRestoreObject) string {
	return fmt.Sprintf("%s-%s", obj.GetName(), obj.GetAction())
}
