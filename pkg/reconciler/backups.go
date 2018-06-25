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
	"strings"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

func (r *AerospikeClusterReconciler) backupCluster(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	// create a backup of each namespace specified in .spec.namespaces
	for _, namespace := range aerospikeCluster.Spec.Namespaces {
		if err := r.createNamespaceBackup(aerospikeCluster, namespace.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *AerospikeClusterReconciler) isClusterBackupFinished(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (bool, error) {
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

func (r *AerospikeClusterReconciler) createNamespaceBackup(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, ns string) error {
	backup := aerospikev1alpha1.AerospikeNamespaceBackup{
		ObjectMeta: v1.ObjectMeta{
			Name: getBackupName(aerospikeCluster, ns),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelClusterKey:   aerospikeCluster.Name,
				selectors.LabelNamespaceKey: ns,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha1.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeClusterKind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Spec: aerospikev1alpha1.AerospikeNamespaceBackupSpec{
			Target: aerospikev1alpha1.TargetNamespace{
				Cluster:   aerospikeCluster.Name,
				Namespace: ns,
			},
			Storage: &aerospikeCluster.Spec.BackupSpec.Storage,
			TTL:     "0d",
		},
	}

	_, err := r.aerospikeclientset.AerospikeV1alpha1().AerospikeNamespaceBackups(aerospikeCluster.Namespace).Create(&backup)
	if err != nil {
		return err
	}
	return nil
}

func (r *AerospikeClusterReconciler) isBackupCompleted(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, ns string) (bool, error) {
	// get the aerospikenamespacebackup resource
	backup, err := r.aerospikeBackupsLister.AerospikeNamespaceBackups(aerospikeCluster.Namespace).Get(getBackupName(aerospikeCluster, ns))
	if err != nil {
		return false, err
	}

	// look for ConditionBackupFinished
	for _, condition := range backup.Status.Conditions {
		if condition.Type == aerospikev1alpha1.ConditionBackupFinished &&
			condition.Status == apiextensions.ConditionTrue {
			return true, nil
		} else if condition.Type == aerospikev1alpha1.ConditionBackupFailed &&
			condition.Status == apiextensions.ConditionTrue {
			return false, errors.ClusterBackupFailed
		}
	}
	return false, nil
}

func getBackupName(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, ns string) string {
	return fmt.Sprintf("%s-%s-%s-upgrade", ns,
		strings.Replace(aerospikeCluster.Status.Version, ".", "", -1),
		strings.Replace(aerospikeCluster.Spec.Version, ".", "", -1),
	)
}
