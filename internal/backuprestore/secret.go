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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/pointers"
	"github.com/travelaudience/aerospike-operator/internal/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/internal/utils/selectors"
)

func (h *AerospikeBackupRestoreHandler) getSecret(obj aerospikev1alpha2.BackupRestoreObject) (*corev1.Secret, error) {
	namespace := obj.GetStorage().GetSecretNamespace(obj.GetNamespace())
	secret, err := h.kubeclientset.CoreV1().Secrets(namespace).Get(obj.GetStorage().GetSecret(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if namespace == obj.GetNamespace() {
		return secret, nil
	}
	return h.createTempSecret(secret, obj)
}

func (h *AerospikeBackupRestoreHandler) clearSecrets(obj aerospikev1alpha2.BackupRestoreObject) error {
	secrets, err := h.kubeclientset.CoreV1().Secrets(obj.GetNamespace()).List(listoptions.ResourcesByBackupRestoreObject(obj))
	if err != nil {
		return err
	}
	for _, secret := range secrets.Items {
		if err := h.kubeclientset.CoreV1().Secrets(secret.Namespace).Delete(secret.Name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (h *AerospikeBackupRestoreHandler) createTempSecret(secret *corev1.Secret, obj aerospikev1alpha2.BackupRestoreObject) (*corev1.Secret, error) {
	return h.kubeclientset.CoreV1().Secrets(obj.GetNamespace()).Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", secret.Name),
			Labels: map[string]string{
				selectors.LabelAppKey:          selectors.LabelAppVal,
				string(obj.GetOperationType()): obj.GetName(),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha2.SchemeGroupVersion.String(),
					Kind:               obj.GetKind(),
					Name:               obj.GetName(),
					UID:                obj.GetObjectMeta().UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
		},
		Data: secret.Data,
		Type: secret.Type,
	})
}
