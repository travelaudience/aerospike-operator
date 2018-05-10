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
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
)

func (h *AerospikeBackupsHandler) addCompletedCondition(obj *BackupRestoreObject) (err error) {
	switch object := obj.Obj.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		if object.Spec.TTL == "" {
			object.Spec.TTL = "0d"
		}
		if object.Status.Conditions == nil {
			object.Status.Conditions = make([]apiextensions.CustomResourceDefinitionCondition, 0)
		}
		object.Status.Conditions = append(object.Status.Conditions, apiextensions.CustomResourceDefinitionCondition{
			Type:   aerospikev1alpha1.ConditionCompleted,
			Status: apiextensions.ConditionTrue,
			LastTransitionTime: v1.Time{
				Time: time.Now(),
			},
		})
		_, err = h.aerospikeclientset.AerospikeV1alpha1().AerospikeNamespaceBackups(object.Namespace).Update(object)
	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		if object.Status.Conditions == nil {
			object.Status.Conditions = make([]apiextensions.CustomResourceDefinitionCondition, 0)
		}
		object.Status.Conditions = append(object.Status.Conditions, apiextensions.CustomResourceDefinitionCondition{
			Type:   aerospikev1alpha1.ConditionCompleted,
			Status: apiextensions.ConditionTrue,
			LastTransitionTime: v1.Time{
				Time: time.Now(),
			},
		})
		_, err = h.aerospikeclientset.AerospikeV1alpha1().AerospikeNamespaceRestores(object.Namespace).Update(object)
	}
	return
}

func (h *AerospikeBackupsHandler) operationAlreadyPerformed(obj *BackupRestoreObject) bool {
	var conditions []apiextensions.CustomResourceDefinitionCondition
	switch object := obj.Obj.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		conditions = object.Status.Conditions
	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		conditions = object.Status.Conditions
	default:
		return false
	}
	if conditions != nil {
		for _, c := range conditions {
			if c.Type == aerospikev1alpha1.ConditionCompleted {
				return c.Status == apiextensions.ConditionTrue
			}
		}
	}
	return false
}
