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
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/errors"
)

func (h *AerospikeBackupsHandler) ensureSecretExists(obj aerospikev1alpha1.BackupRestoreObject) error {
	secret, err := h.secretsLister.Secrets(obj.GetObjectMeta().Namespace).Get(obj.GetStorage().Secret)
	if err != nil {
		return err
	}

	if _, ok := secret.Data[secretFileName]; ok {
		return nil
	}
	return errors.InvalidSecretFileName
}
