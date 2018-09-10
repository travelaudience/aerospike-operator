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

package framework

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
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

func (tf *TestFramework) NewAerospikeNamespaceBackupGCS(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, ttl *string) aerospikev1alpha2.AerospikeNamespaceBackup {
	return aerospikev1alpha2.AerospikeNamespaceBackup{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: backupPrefix,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: &aerospikev1alpha1.BackupStorageSpec{
				Type:   aerospikev1alpha1.StorageTypeGCS,
				Bucket: GCSBucketName,
				Secret: GCSSecretName,
			},
			TTL: ttl,
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceRestoreGCS(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, backup *aerospikev1alpha2.AerospikeNamespaceBackup) aerospikev1alpha2.AerospikeNamespaceRestore {
	return aerospikev1alpha2.AerospikeNamespaceRestore{
		ObjectMeta: v1.ObjectMeta{
			Name: backup.Name,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceRestoreSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: &aerospikev1alpha1.BackupStorageSpec{
				Type:   aerospikev1alpha1.StorageTypeGCS,
				Bucket: GCSBucketName,
				Secret: GCSSecretName,
			},
		},
	}
}

func (tf *TestFramework) WaitForBackupRestoreCondition(obj aerospikev1alpha2.BackupRestoreObject, fn watch.ConditionFunc, timeout time.Duration) (err error) {
	var w watch.Interface
	switch obj.GetOperationType() {
	case common.OperationTypeBackup:
		w, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(obj.GetNamespace()).Watch(listoptions.ObjectByName(obj.GetName()))
	case common.OperationTypeRestore:
		w, err = tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(obj.GetNamespace()).Watch(listoptions.ObjectByName(obj.GetName()))
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

func (tf *TestFramework) WaitForBackupRestoreCompleted(obj aerospikev1alpha2.BackupRestoreObject) error {
	return tf.WaitForBackupRestoreCondition(obj, func(event watch.Event) (bool, error) {
		obj := event.Object.(aerospikev1alpha2.BackupRestoreObject)
		conditions := obj.GetConditions()
		if conditions != nil {
			for _, c := range conditions {
				if c.Type == obj.GetFinishedConditionType() {
					return true, nil
				}
			}
		}
		return false, nil
	}, watchTimeout)
}
