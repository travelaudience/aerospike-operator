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

package backuphandler

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/reconciler"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

func (h *AerospikeBackupsHandler) createJob(obj aerospikev1alpha1.BackupRestoreObject) error {
	job := v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s-job", obj.GetObjectMeta().Name, obj.GetAction()),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelClusterKey:   obj.GetTarget().Cluster,
				selectors.LabelNamespaceKey: obj.GetTarget().Namespace,
				obj.GetType():               obj.GetObjectMeta().Name,
			},
			Namespace: obj.GetObjectMeta().Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeBackupKind,
					Name:               obj.GetObjectMeta().Name,
					UID:                obj.GetObjectMeta().UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(obj.GetAction()),
					Namespace: obj.GetObjectMeta().Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "aerospike-operator-tools",
							Image:           "quay.io/travelaudience/aerospike-operator-tools:dev-testing",
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"backup",
								backupCommand(obj.GetAction()),
								"-bucket-name", obj.GetStorage().Bucket,
								"-name", fmt.Sprintf("%s.%s", obj.GetObjectMeta().Name, backupExtension),
								"-data-pipe-path", fmt.Sprintf("%s/%s", sharedVolumeMountPath, sharedDataPipeName),
								"-meta-pipe-path", fmt.Sprintf("%s/%s", sharedVolumeMountPath, sharedMetadataPipeName),
								"-secret-path", fmt.Sprintf("%s/%s", bucketSecretVolumeMountPath, secretFileName),
								"-debug",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      bucketSecretVolumeName,
									ReadOnly:  true,
									MountPath: bucketSecretVolumeMountPath,
								},
								{
									Name:      sharedVolumeName,
									MountPath: sharedVolumeMountPath,
								},
							},
						},
						{
							Name:            "aerospike-tools",
							Image:           "aerospike/aerospike-tools:3.15.3.6",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/bash", "-c",
							},
							Args: []string{
								fmt.Sprintf("%s && %s -h %s -p %d -n %s %s - %s %s",
									metaCommand(obj.GetAction(), obj.GetTarget().Namespace),
									fmt.Sprintf("as%s", obj.GetAction()),
									obj.GetTarget().Cluster,
									reconciler.ServicePort,
									getNamespace(obj.GetAction(), obj.GetTarget().Namespace),
									inputOutputString(obj.GetAction()),
									pipeDirection(obj.GetAction()), fmt.Sprintf("%s/%s", sharedVolumeMountPath, sharedDataPipeName),
								),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      sharedVolumeName,
									MountPath: sharedVolumeMountPath,
								},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:            "init-pipe",
							Image:           "busybox",
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/bin/sh", "-c",
							},
							Args: []string{
								fmt.Sprintf("%s && %s",
									fmt.Sprintf("mkfifo %s/%s", sharedVolumeMountPath, sharedDataPipeName),
									fmt.Sprintf("mkfifo %s/%s", sharedVolumeMountPath, sharedMetadataPipeName),
								),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      sharedVolumeName,
									MountPath: sharedVolumeMountPath,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Volumes: []corev1.Volume{
						{
							Name: bucketSecretVolumeName,
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: obj.GetStorage().Secret,
								},
							},
						},
						{
							Name: sharedVolumeName,
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	res, err := h.kubeclientset.BatchV1().Jobs(obj.GetObjectMeta().Namespace).Create(&job)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"job": meta.Key(res),
	}).Debugf("%s job created", obj.GetAction())
	return nil
}

func (h *AerospikeBackupsHandler) getJobStatus(obj aerospikev1alpha1.BackupRestoreObject) (*v1.JobStatus, error) {
	jobs, err := h.jobsLister.Jobs(obj.GetObjectMeta().Namespace).List(labels.SelectorFromSet(map[string]string{
		selectors.LabelAppKey:       selectors.LabelAppVal,
		selectors.LabelClusterKey:   obj.GetTarget().Cluster,
		selectors.LabelNamespaceKey: obj.GetTarget().Namespace,
		obj.GetType():               obj.GetObjectMeta().Name,
	}))
	if err != nil {
		return nil, err
	}
	if len(jobs) > 0 {
		return &jobs[0].Status, nil
	}
	log.WithFields(log.Fields{
		obj.GetType(): meta.Key(obj),
	}).Debugf("%s job does not exist", obj.GetAction())
	return nil, errors.JobDoesNotExist
}
