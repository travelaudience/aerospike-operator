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

package v1alpha2

import (
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
)

const (
	// convertedFromAnnotationKey is the annotation set on resources that have been automatically converted. this avoids
	// unnecessary repeated conversion.
	convertedFromAnnotationKey = "aerospike.travelaudience.com/converted-from"
)

// ConvertResources lists and converts existing resources and converts them to v1alpha2.
func ConvertResources(extsClient *extsclientset.Clientset, aerospikeClient *aerospikeclientset.Clientset) error {
	// convert existing aerospikecluster resources
	if err := convertAerospikeClusters(extsClient, aerospikeClient); err != nil {
		return err
	}
	// convert existing aerospikenamespacebackup resources
	if err := convertAerospikeNamespaceBackups(extsClient, aerospikeClient); err != nil {
		return err
	}
	// convert existing aerospikenamespacerestore resources
	if err := convertAerospikeNamespaceRestores(extsClient, aerospikeClient); err != nil {
		return err
	}
	return nil
}
