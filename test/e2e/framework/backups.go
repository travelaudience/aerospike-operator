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
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

const (
	backupPrefix = "as-backup-e2e-"
)

var (
	GCSBucketName      string
	GCSSecretName      string
	GCSSecretNamespace string
	GCSSecretKey       string
)

func (tf *TestFramework) NewAerospikeNamespaceBackupGCS(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, ttl *string) aerospikev1alpha2.AerospikeNamespaceBackup {
	return aerospikev1alpha2.AerospikeNamespaceBackup{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupPrefix,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: &aerospikev1alpha2.BackupStorageSpec{
				Type:            common.StorageTypeGCS,
				Bucket:          GCSBucketName,
				Secret:          GCSSecretName,
				SecretNamespace: &GCSSecretNamespace,
				SecretKey:       &GCSSecretKey,
			},
			TTL: ttl,
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceBackupGCSWithoutBackupStorageSpec(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, ttl *string) aerospikev1alpha2.AerospikeNamespaceBackup {
	return aerospikev1alpha2.AerospikeNamespaceBackup{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: backupPrefix,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			TTL: ttl,
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceRestoreGCS(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, backup *aerospikev1alpha2.AerospikeNamespaceBackup) aerospikev1alpha2.AerospikeNamespaceRestore {
	return aerospikev1alpha2.AerospikeNamespaceRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name: backup.Name,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceRestoreSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
			Storage: &aerospikev1alpha2.BackupStorageSpec{
				Type:            common.StorageTypeGCS,
				Bucket:          GCSBucketName,
				Secret:          GCSSecretName,
				SecretNamespace: &GCSSecretNamespace,
				SecretKey:       &GCSSecretKey,
			},
		},
	}
}

func (tf *TestFramework) NewAerospikeNamespaceRestoreGCSWithoutBackupStorageSpec(cluster *aerospikev1alpha2.AerospikeCluster, namespace string, backup *aerospikev1alpha2.AerospikeNamespaceBackup) aerospikev1alpha2.AerospikeNamespaceRestore {
	return aerospikev1alpha2.AerospikeNamespaceRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name: backup.Name,
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceRestoreSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   cluster.Name,
				Namespace: namespace,
			},
		},
	}
}

func (tf *TestFramework) WaitForBackupRestoreCondition(obj aerospikev1alpha2.BackupRestoreObject, fn watch.ConditionFunc, timeout time.Duration) (err error) {
	var lw *cache.ListWatch
	var lt runtime.Object
	fs := selectors.ObjectByCoordinates(obj.GetNamespace(), obj.GetName())
	switch obj.GetOperationType() {
	case common.OperationTypeBackup:
		lw = &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fs.String()
				return tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(obj.GetNamespace()).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watchapi.Interface, error) {
				options.FieldSelector = fs.String()
				return tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceBackups(obj.GetNamespace()).Watch(options)
			},
		}
		lt = &aerospikev1alpha2.AerospikeNamespaceBackup{}
	case common.OperationTypeRestore:
		lw = &cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fs.String()
				return tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(obj.GetNamespace()).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watchapi.Interface, error) {
				options.FieldSelector = fs.String()
				return tf.AerospikeClient.AerospikeV1alpha2().AerospikeNamespaceRestores(obj.GetNamespace()).Watch(options)
			},
		}
		lt = &aerospikev1alpha2.AerospikeNamespaceRestore{}
	}
	ctx, cfn := context.WithTimeout(context.Background(), timeout)
	defer cfn()
	last, err := watch.UntilWithSync(ctx, lw, lt, nil, fn)
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for object %q", meta.Key(obj))
	}
	return nil
}

func (tf *TestFramework) WaitForBackupRestoreCompleted(obj aerospikev1alpha2.BackupRestoreObject) error {
	return tf.WaitForBackupRestoreCondition(obj, func(event watchapi.Event) (bool, error) {
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
