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
	"k8s.io/apimachinery/pkg/fields"
)

// ObjectByName returns a selector that matches an object by its name.
func ObjectByName(name string) fields.Selector {
	set := map[string]string{
		"metadata.name": name,
	}
	return fields.SelectorFromSet(set)
}

// ObjectByCoordinates returns a selector that matches an object by its namespace and name.
func ObjectByCoordinates(namespace, name string) fields.Selector {
	return fields.SelectorFromSet(map[string]string{
		"metadata.namespace": namespace,
		"metadata.name":      name,
	})
}
