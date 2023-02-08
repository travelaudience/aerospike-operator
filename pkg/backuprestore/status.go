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
	"context"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
)

func (h *AerospikeBackupRestoreHandler) updateStatus(obj aerospikev1alpha2.BackupRestoreObject) error {
	var err error
	switch obj.GetOperationType() {
	case common.OperationTypeBackup:
		_, err = h.aerospikeclientset.AerospikeV1alpha2().AerospikeNamespaceBackups(obj.GetNamespace()).UpdateStatus(context.TODO(), obj.(*aerospikev1alpha2.AerospikeNamespaceBackup), v1.UpdateOptions{})
	case common.OperationTypeRestore:
		_, err = h.aerospikeclientset.AerospikeV1alpha2().AerospikeNamespaceRestores(obj.GetNamespace()).UpdateStatus(context.TODO(), obj.(*aerospikev1alpha2.AerospikeNamespaceRestore), v1.UpdateOptions{})
	}
	return err
}

func (h *AerospikeBackupRestoreHandler) isFailedOrFinished(obj aerospikev1alpha2.BackupRestoreObject) bool {
	for _, c := range obj.GetConditions() {
		if (c.Type == obj.GetFinishedConditionType() || c.Type == obj.GetFailedConditionType()) && c.Status == apiextensions.ConditionTrue {
			return true
		}
	}
	return false
}
