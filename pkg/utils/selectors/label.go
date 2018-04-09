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
	"k8s.io/apimachinery/pkg/labels"
)

const (
	// LabelAppKey represents the name of the "app" label added to every pod.
	LabelAppKey = "app"
	// LabelAppVal represents the value of the "app" label added to every pod.
	LabelAppVal = "aerospike"
	// LabelClusterKey respresents the name of the "cluster" label added to every pod.
	LabelClusterKey = "cluster"
)

// PodsByClusterName returns a selector that matches all pods belonging to a given AerospikeCluster.
func PodsByClusterName(name string) labels.Selector {
	set := map[string]string{
		LabelAppKey:     LabelAppVal,
		LabelClusterKey: name,
	}
	return labels.SelectorFromSet(set)
}
