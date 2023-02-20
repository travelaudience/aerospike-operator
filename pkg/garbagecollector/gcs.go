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

package garbagecollector

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/backuprestore"
	"github.com/travelaudience/aerospike-operator/pkg/backuprestore/gcs"
)

func (h *AerospikeNamespaceBackupHandler) deleteBackupDataGCS(asBackup *aerospikev1alpha2.AerospikeNamespaceBackup) error {
	// get the secret containing the credentials to access the gcs bucket
	namespace := asBackup.Spec.Storage.GetSecretNamespace(asBackup.Namespace)
	secret, err := h.kubeclientset.CoreV1().Secrets(namespace).Get(context.TODO(), asBackup.Spec.Storage.GetSecret(), v1.GetOptions{})
	if err != nil {
		return err
	}
	// get gcs client
	client, err := gcs.NewGCSClientFromJSON(secret.Data[asBackup.Spec.Storage.GetSecretKey()])
	if err != nil {
		return err
	}
	defer client.Close()

	err = client.DeleteObject(asBackup.Spec.Storage.Bucket, backuprestore.GetMetadataObjectName(asBackup.Name))
	if err != nil {
		return err
	}
	return client.DeleteObject(asBackup.Spec.Storage.Bucket, backuprestore.GetBackupObjectName(asBackup.Name))
}
