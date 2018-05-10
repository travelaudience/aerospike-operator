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
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

func (h *AerospikeBackupsHandler) createJob(obj *BackupRestoreObject) error {
	job := v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-job-", obj.Name, obj.Action),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelClusterKey:   obj.Target.Cluster,
				selectors.LabelNamespaceKey: obj.Target.Namespace,
				obj.Type:                    obj.Name,
			},
			Namespace: obj.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         v1alpha1.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeBackupKind,
					Name:               obj.Name,
					UID:                obj.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: v1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(obj.Action),
					Namespace: obj.Namespace,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "aerospike-operator-tools",
							Image:           "quay.io/travelaudience/aerospike-operator-tools:dev",
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"backup",
								fmt.Sprintf("-%s", obj.Action),
								"-host", obj.Target.Cluster,
								"-port", "3000",
								"-namespace", obj.Target.Namespace,
								"-bucket", obj.Storage.Bucket,
								"-name", fmt.Sprintf("%s.%s", obj.Name, backupExtension),
								"-debug",
								"-compress",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      bucketSecretVolumeName,
									ReadOnly:  true,
									MountPath: bucketSecretVolumeMountPath,
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
									SecretName: obj.Storage.Secret,
								},
							},
						},
					},
				},
			},
		},
	}

	// create the job
	res, err := h.kubeclientset.BatchV1().Jobs(obj.Namespace).Create(&job)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"job": meta.Key(res),
	}).Debugf("%s job created", obj.Action)

	// wait for job to complete
	err = h.waitForJobCondition(res, watchJobTimeout, func(event watch.Event) (exit bool, err error) {
		job := event.Object.(*v1.Job)
		if job.Status.Failed > 0 {
			err = errors.JobFailed
			exit = true
		}
		exit = job.Status.Succeeded == 1
		return
	})
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"job": meta.Key(res),
	}).Debugf("%s job completed", obj.Action)
	return nil
}

func (h *AerospikeBackupsHandler) ensureJobDoesNotExist(obj *BackupRestoreObject) error {
	jobs, err := h.jobsLister.Jobs(obj.Namespace).List(labels.SelectorFromSet(map[string]string{
		selectors.LabelAppKey:       selectors.LabelAppVal,
		selectors.LabelClusterKey:   obj.Target.Cluster,
		selectors.LabelNamespaceKey: obj.Target.Namespace,
		obj.Type:                    obj.Name,
	}))
	if err != nil {
		return err
	}
	if len(jobs) > 0 {
		log.WithFields(log.Fields{
			obj.Type: meta.Key(obj.Obj),
		}).Debugf("%s job is already running", obj.Action)
		return errors.JobAlreadyExists
	}
	return nil
}
