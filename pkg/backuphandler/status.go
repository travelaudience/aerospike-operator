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
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
)

func (h *AerospikeBackupsHandler) getConditionStatus(obj aerospikev1alpha1.BackupRestoreObject, conditionType apiextensions.CustomResourceDefinitionConditionType) apiextensions.ConditionStatus {
	var conditions []apiextensions.CustomResourceDefinitionCondition
	switch object := obj.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(object.Status.Conditions))
		copy(conditions, object.Status.Conditions)
	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(object.Status.Conditions))
		copy(conditions, object.Status.Conditions)
	}
	if conditions != nil {
		for _, c := range conditions {
			if c.Type == conditionType {
				return c.Status
			}
		}
	}
	return apiextensions.ConditionUnknown
}

func (h *AerospikeBackupsHandler) setConditions(obj aerospikev1alpha1.BackupRestoreObject, conditionsMap map[apiextensions.CustomResourceDefinitionConditionType]apiextensions.ConditionStatus) error {
	var conditions []apiextensions.CustomResourceDefinitionCondition

	switch object := obj.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(object.Status.Conditions))
		copy(conditions, object.Status.Conditions)
	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(object.Status.Conditions))
		copy(conditions, object.Status.Conditions)
	default:
		return fmt.Errorf("unsupported type")
	}

	for t, s := range conditionsMap {
		exists := false
		for i, c := range conditions {
			if c.Type == t {
				conditions[i].Status = s
				conditions[i].LastTransitionTime = v1.Time{
					Time: time.Now(),
				}
				exists = true
				break
			}
		}
		if !exists {
			conditions = append(conditions, apiextensions.CustomResourceDefinitionCondition{
				Type:   t,
				Status: s,
				LastTransitionTime: v1.Time{
					Time: time.Now(),
				},
			})
		}
	}

	switch object := obj.(type) {
	case *aerospikev1alpha1.AerospikeNamespaceBackup:
		oldBytes, err := json.Marshal(object)
		if err != nil {
			return err
		}
		object.Status.Conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(conditions))
		copy(object.Status.Conditions, conditions)
		newBytes, err := json.Marshal(object)
		if err != nil {
			return err
		}
		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, &aerospikev1alpha1.AerospikeNamespaceBackup{})
		if err != nil {
			return err
		}
		if _, err = h.aerospikeclientset.AerospikeV1alpha1().AerospikeNamespaceBackups(object.Namespace).Patch(object.Name, types.MergePatchType, patchBytes); err != nil {
			return err
		}
	case *aerospikev1alpha1.AerospikeNamespaceRestore:
		oldBytes, err := json.Marshal(object)
		if err != nil {
			return err
		}
		object.Status.Conditions = make([]apiextensions.CustomResourceDefinitionCondition, len(conditions))
		copy(object.Status.Conditions, conditions)
		newBytes, err := json.Marshal(object)
		if err != nil {
			return err
		}
		patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, &aerospikev1alpha1.AerospikeNamespaceRestore{})
		if err != nil {
			return err
		}
		if _, err = h.aerospikeclientset.AerospikeV1alpha1().AerospikeNamespaceRestores(object.Namespace).Patch(object.Name, types.MergePatchType, patchBytes); err != nil {
			return err
		}
	}
	return nil
}
