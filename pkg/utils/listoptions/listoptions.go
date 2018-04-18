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

package listoptions

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

// ObjectByField returns the options for a list/watch operation that searches for an object based on
// the value of the given field.
func ObjectByField(name, value string) metav1.ListOptions {
	return metav1.ListOptions{
		FieldSelector: fmt.Sprintf("%s=%s", name, value),
	}
}

// ObjectByName returns the options for a list/watch operation that searches for an object based on
// its name.
func ObjectByName(name string) metav1.ListOptions {
	return metav1.ListOptions{
		FieldSelector: selectors.ObjectByName(name).String(),
	}
}

// PodsByClusterName returns the options for a list/watch operation that searches for pods belonging
// to a given AerospikeCluster.
func PodsByClusterName(name string) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: selectors.PodsByClusterName(name).String(),
	}
}
