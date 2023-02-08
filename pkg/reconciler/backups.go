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
	"context"
	"fmt"
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

func (r *AerospikeClusterReconciler) backupCluster(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) error {
	// create a backup of each namespace specified in .spec.namespaces
	for _, namespace := range aerospikeCluster.Spec.Namespaces {
		if err := r.createNamespaceBackup(aerospikeCluster, namespace.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *AerospikeClusterReconciler) isClusterBackupFinished(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) (bool, error) {
	// if the backup of one of the namespaces have not finished, return false
	for _, namespace := range aerospikeCluster.Spec.Namespaces {
		if finished, err := r.isBackupCompleted(aerospikeCluster, namespace.Name); err != nil {
			return false, err
		} else if !finished {
			return false, nil
		}
	}
	return true, nil
}

func (r *AerospikeClusterReconciler) createNamespaceBackup(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, ns string) error {
	backup := aerospikev1alpha2.AerospikeNamespaceBackup{
		ObjectMeta: v1.ObjectMeta{
			Name: GetBackupName(ns, aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelClusterKey:   aerospikeCluster.Name,
				selectors.LabelNamespaceKey: ns,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha2.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeClusterKind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: aerospikev1alpha2.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha2.TargetNamespace{
				Cluster:   aerospikeCluster.Name,
				Namespace: ns,
			},
			Storage: &aerospikev1alpha2.BackupStorageSpec{
				Type:            aerospikeCluster.Spec.BackupSpec.Storage.Type,
				Bucket:          aerospikeCluster.Spec.BackupSpec.Storage.Bucket,
				Secret:          aerospikeCluster.Spec.BackupSpec.Storage.GetSecret(),
				SecretNamespace: aerospikeCluster.Spec.BackupSpec.Storage.SecretNamespace,
				SecretKey:       aerospikeCluster.Spec.BackupSpec.Storage.SecretKey,
			},
			TTL: aerospikeCluster.Spec.BackupSpec.TTL,
		},
	}

	_, err := r.aerospikeclientset.AerospikeV1alpha2().AerospikeNamespaceBackups(aerospikeCluster.Namespace).Create(context.TODO(), &backup, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (r *AerospikeClusterReconciler) isBackupCompleted(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, ns string) (bool, error) {
	// get the AerospikeNamespaceBackup resource
	backup, err := r.aerospikeBackupsLister.AerospikeNamespaceBackups(aerospikeCluster.Namespace).Get(GetBackupName(ns, aerospikeCluster.Status.Version, aerospikeCluster.Spec.Version))
	if err != nil {
		return false, err
	}

	// look for ConditionBackupFinished
	for _, condition := range backup.Status.Conditions {
		if condition.Type == common.ConditionBackupFinished &&
			condition.Status == apiextensions.ConditionTrue {
			return true, nil
		} else if condition.Type == common.ConditionBackupFailed &&
			condition.Status == apiextensions.ConditionTrue {
			return false, errors.ClusterBackupFailed
		}
	}
	return false, nil
}

// GetBackupName returns the name of a backup created automatically before upgrading
func GetBackupName(ns, sourceVersion, targetVersion string) string {
	return fmt.Sprintf("%s-%s-%s-upgrade", ns,
		strings.Replace(sourceVersion, ".", "", -1),
		strings.Replace(targetVersion, ".", "", -1),
	)
}
