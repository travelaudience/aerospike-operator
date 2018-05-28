/*
Copyright 2018 The aerospike-controller Authors.

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

package framework

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
)

const (
	backupPrefix = "as-backup-e2e-"
)

var (
	GCSBucketName string
	GCSSecretName string
)

func (tf *TestFramework) CopySecretToNamespace(secret string, ns *corev1.Namespace) error {
	s, err := tf.KubeClient.CoreV1().Secrets(v1.NamespaceDefault).Get(secret, v1.GetOptions{})
	if err != nil {
		return err
	}
	s.Namespace = ns.Name
	s.ResourceVersion = ""
	s.UID = ""
	_, err = tf.KubeClient.CoreV1().Secrets(ns.Name).Create(s)
	return err
}

func (tf *TestFramework) NewAerospikeNamespaceBackupGCS(cluster *aerospikev1alpha1.AerospikeCluster, namespace, ttl string) aerospikev1alpha1.AerospikeNamespaceBackup {
	return aerospikev1alpha1.AerospikeNamespaceBackup{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: backupPrefix,
		},
		Spec: aerospikev1alpha1.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha1.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: aerospikev1alpha1.BackupStorageSpec{
				Type:   aerospikev1alpha1.StorageTypeGCS,
				Bucket: GCSBucketName,
				Secret: GCSSecretName,
			},
			TTL: ttl,
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceRestoreGCS(cluster *aerospikev1alpha1.AerospikeCluster, namespace string, backup *aerospikev1alpha1.AerospikeNamespaceBackup) aerospikev1alpha1.AerospikeNamespaceRestore {
	return aerospikev1alpha1.AerospikeNamespaceRestore{
		ObjectMeta: v1.ObjectMeta{
			Name: backup.Name,
		},
		Spec: aerospikev1alpha1.AerospikeNamespaceRestoreSpec{
			Target: aerospikev1alpha1.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: aerospikev1alpha1.BackupStorageSpec{
				Type:   aerospikev1alpha1.StorageTypeGCS,
				Bucket: GCSBucketName,
				Secret: GCSSecretName,
			},
		},
	}
}

func (tf *TestFramework) WaitForBackupRestoreCondition(obj aerospikev1alpha1.BackupRestoreObject, fn watch.ConditionFunc, timeout time.Duration) (err error) {
	var w watch.Interface
	switch obj.GetAction() {
	case aerospikev1alpha1.ActionTypeBackup:
		w, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeNamespaceBackups(obj.GetNamespace()).Watch(listoptions.ObjectByName(obj.GetName()))
	case aerospikev1alpha1.ActionTypeRestore:
		w, err = tf.AerospikeClient.AerospikeV1alpha1().AerospikeNamespaceRestores(obj.GetNamespace()).Watch(listoptions.ObjectByName(obj.GetName()))
	}
	if err != nil {
		return err
	}
	start := time.Now()
	last, err := watch.Until(timeout, w, fn)
	if err != nil {
		if err == watch.ErrWatchClosed {
			if t := timeout - time.Since(start); t > 0 {
				return tf.WaitForBackupRestoreCondition(obj, fn, t)
			}
		}
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for %s", meta.Key(obj))
	}
	return nil
}

func (tf *TestFramework) WaitForBackupRestoreCompleted(obj aerospikev1alpha1.BackupRestoreObject) error {
	return tf.WaitForBackupRestoreCondition(obj, func(event watch.Event) (bool, error) {
		obj := event.Object.(aerospikev1alpha1.BackupRestoreObject)
		conditions := obj.GetConditions()
		if conditions != nil {
			for _, c := range conditions {
				if c.Type == aerospikev1alpha1.ConditionCompleted {
					return true, nil
				}
			}
		}
		return false, nil
	}, watchTimeout)
}
