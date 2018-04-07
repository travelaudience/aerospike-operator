/*
Copyright 2018 The aerospike-controller Authors.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	aslabels "github.com/travelaudience/aerospike-operator/pkg/utils/labels"
)

// ClusterByName returns the options for a list/watch operation that searches for a given AerospikeCluster by its name.
func ClusterByName(name string) metav1.ListOptions {
	set := map[string]string{
		"metadata.name": name,
	}
	return metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(set).String(),
	}
}

// PodsByClusterName returns the options for a list/watch operation that searches for pods belonging to a given
// AerospikeCluster.
func PodsByClusterName(name string) metav1.ListOptions {
	set := map[string]string{
		aslabels.LabelAppKey:     aslabels.LabelAppVal,
		aslabels.LabelClusterKey: name,
	}
	return metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(set).String(),
	}
}
