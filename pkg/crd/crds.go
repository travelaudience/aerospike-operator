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

package crd

import (
	"fmt"

	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	asstrings "github.com/travelaudience/aerospike-operator/pkg/utils/strings"
)

var (
	crds = []*extsv1beta1.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikeclusters", aerospikev1alpha1.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
				Version: aerospikev1alpha1.SchemeGroupVersion.Version,
				Scope:   extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural: "aerospikeclusters",
					Kind:   "AerospikeCluster",
				},
				Validation: &extsv1beta1.CustomResourceValidation{
					OpenAPIV3Schema: &extsv1beta1.JSONSchemaProps{
						Properties: map[string]extsv1beta1.JSONSchemaProps{
							"spec": {
								Properties: map[string]extsv1beta1.JSONSchemaProps{
									"nodeCount": {
										Type:    "integer",
										Maximum: pointers.NewFloat64(8),
										Minimum: pointers.NewFloat64(1),
									},
									"version": {
										Type: "string",
										Enum: []extsv1beta1.JSON{
											{Raw: []byte(asstrings.DoubleQuoted("4.0.0.4"))},
										},
									},
									"namespaces": {
										Type:     "array",
										MaxItems: pointers.NewInt64(2),
										MinItems: pointers.NewInt64(1),
										Items: &extsv1beta1.JSONSchemaPropsOrArray{
											Schema: &extsv1beta1.JSONSchemaProps{
												Title: "namespace",
												Type:  "object",
												Properties: map[string]extsv1beta1.JSONSchemaProps{
													"name": {
														Type:      "string",
														MinLength: pointers.NewInt64(1),
													},
													"replicationFactor": {
														Type:    "integer",
														Minimum: pointers.NewFloat64(1),
														Maximum: pointers.NewFloat64(8),
													},
													"memorySize": {
														Type:    "string",
														Pattern: `^\d+G$`,
													},
													"defaultTTL": {
														Type:    "string",
														Pattern: `^\d+s$`,
													},
													"storage": {
														Type: "object",
														Properties: map[string]extsv1beta1.JSONSchemaProps{
															"type": {
																Type: "string",
																Enum: []extsv1beta1.JSON{
																	{Raw: []byte(asstrings.DoubleQuoted(aerospikev1alpha1.StorageTypeFile))},
																	{Raw: []byte(asstrings.DoubleQuoted(aerospikev1alpha1.StorageTypeDevice))},
																},
															},
															"size": {
																Type:    "string",
																Pattern: `^(20{3}|1\d{1,3}|[1-9])G$`,
															},
														},
														Required: []string{
															"type",
															"size",
														},
													},
												},
												Required: []string{
													"name",
													"storage",
												},
											},
										},
									},
								},
								Required: []string{
									"nodeCount",
									"version",
									"namespaces",
								},
							},
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacebackups", aerospikev1alpha1.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
				Version: aerospikev1alpha1.SchemeGroupVersion.Version,
				Scope:   extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural: "aerospikenamespacebackups",
					Kind:   "AerospikeNamespaceBackup",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacerestores", aerospikev1alpha1.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
				Version: aerospikev1alpha1.SchemeGroupVersion.Version,
				Scope:   extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural: "aerospikenamespacerestores",
					Kind:   "AerospikeNamespaceRestore",
				},
			},
		},
	}
)
