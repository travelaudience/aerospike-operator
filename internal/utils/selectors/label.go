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

package selectors

import (
	"k8s.io/apimachinery/pkg/labels"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
)

const (
	// LabelAppKey represents the name of the "app" label added to every pod.
	LabelAppKey = "app"
	// LabelAppVal represents the value of the "app" label added to every pod.
	LabelAppVal = "aerospike"
	// LabelClusterKey respresents the name of the "cluster" label added to every pod.
	LabelClusterKey = "cluster"
	// LabelNamespaceKey represents the name of the "namespace" label added to every persistent volume claim.
	LabelNamespaceKey = "namespace"
)

// ResourcesByClusterName returns a selector that matches all resources belonging to a given AerospikeCluster.
func ResourcesByClusterName(name string) labels.Selector {
	set := map[string]string{
		LabelAppKey:     LabelAppVal,
		LabelClusterKey: name,
	}
	return labels.SelectorFromSet(set)
}

// ResourcesByBackupRestoreObject returns a selector that matches all resources belonging to a given BackupRestoreObject.
func ResourcesByBackupRestoreObject(obj aerospikev1alpha2.BackupRestoreObject) labels.Selector {
	set := map[string]string{
		LabelAppKey:                    LabelAppVal,
		string(obj.GetOperationType()): obj.GetName(),
	}
	return labels.SelectorFromSet(set)
}
