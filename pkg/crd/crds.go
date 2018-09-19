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

package crd

import (
	"fmt"

	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	asstrings "github.com/travelaudience/aerospike-operator/pkg/utils/strings"
)

const (
	AerospikeClusterKind   = common.AerospikeClusterKind
	AerospikeClusterPlural = "aerospikeclusters"
	AerospikeClusterShort  = "asc"

	AerospikeNamespaceBackupKind   = common.AerospikeNamespaceBackupKind
	AerospikeNamespaceBackupPlural = "aerospikenamespacebackups"
	AerospikeNamespaceBackupShort  = "asnb"

	AerospikeNamespaceRestoreKind   = common.AerospikeNamespaceRestoreKind
	AerospikeNamespaceRestorePlural = "aerospikenamespacerestores"
	AerospikeNamespaceRestoreShort  = "asnr"

	// ttlPattern is the regex used to match a number of days (with
	// optional fraction) suffixed with a "d"
	ttlPattern = `^([0-9]*[.])?[0-9]+d$`
)

var (
	AerospikeClusterCRDName          = fmt.Sprintf("%s.%s", AerospikeClusterPlural, aerospikev1alpha2.SchemeGroupVersion.Group)
	AerospikeNamespaceBackupCRDName  = fmt.Sprintf("%s.%s", AerospikeNamespaceBackupPlural, aerospikev1alpha2.SchemeGroupVersion.Group)
	AerospikeNamespaceRestoreCRDName = fmt.Sprintf("%s.%s", AerospikeNamespaceRestorePlural, aerospikev1alpha2.SchemeGroupVersion.Group)
)

var (
	backupStorageSpecProps = extsv1beta1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]extsv1beta1.JSONSchemaProps{
			"type": {
				Type: "string",
				Enum: []extsv1beta1.JSON{
					{Raw: []byte(asstrings.DoubleQuoted(common.StorageTypeGCS))},
				},
			},
			"bucket": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
			"secret": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
			"secretNamespace": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
			"secretKey": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
		},
		Required: []string{
			"type",
			"bucket",
			"secret",
		},
	}

	backupRestoreTargetProps = extsv1beta1.JSONSchemaProps{
		Type: "object",
		Properties: map[string]extsv1beta1.JSONSchemaProps{
			"cluster": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
			"namespace": {
				Type:      "string",
				MinLength: pointers.NewInt64(1),
			},
		},
		Required: []string{
			"cluster",
			"namespace",
		},
	}

	crds = []*extsv1beta1.CustomResourceDefinition{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: AerospikeClusterCRDName,
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group: aerospikev1alpha2.SchemeGroupVersion.Group,
				Versions: []extsv1beta1.CustomResourceDefinitionVersion{
					{
						Name:    aerospikev1alpha2.SchemeGroupVersion.Version,
						Served:  true,
						Storage: true,
					},
					{
						Name:    aerospikev1alpha1.SchemeGroupVersion.Version,
						Served:  true,
						Storage: false,
					},
				},
				Scope: extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural:     AerospikeClusterPlural,
					Kind:       AerospikeClusterKind,
					ShortNames: []string{AerospikeClusterShort},
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
										Type:    "string",
										Pattern: `^\d+\.\d+\.\d+(\.\d+)?$`,
									},
									"namespaces": {
										Type: "array",
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
																	{Raw: []byte(asstrings.DoubleQuoted(common.StorageTypeFile))},
																	{Raw: []byte(asstrings.DoubleQuoted(common.StorageTypeDevice))},
																},
															},
															"size": {
																Type:    "string",
																Pattern: `^(20{3}|1?\d{1,3}|[1-9])G$`,
															},
															"storageClassName": {
																Type: "string",
															},
															"persistentVolumeClaimTTL": {
																Type:    "string",
																Pattern: ttlPattern,
															},
															"dataInMemory": {
																Type: "boolean",
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
									"backupSpec": {
										Type: "object",
										Properties: map[string]extsv1beta1.JSONSchemaProps{
											"ttl": {
												Type:    "string",
												Pattern: ttlPattern,
											},
											"storage": backupStorageSpecProps,
										},
										Required: []string{
											"storage",
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
				Subresources: &extsv1beta1.CustomResourceSubresources{
					Status: &extsv1beta1.CustomResourceSubresourceStatus{},
					Scale: &extsv1beta1.CustomResourceSubresourceScale{
						SpecReplicasPath:   ".spec.nodeCount",
						StatusReplicasPath: ".status.nodeCount",
						LabelSelectorPath:  nil,
					},
				},
				AdditionalPrinterColumns: []extsv1beta1.CustomResourceColumnDefinition{
					{
						Name:        "Version",
						Type:        "string",
						Description: "The Aerospike version running in the Aerospike cluster",
						JSONPath:    ".status.version",
					},
					{
						Name:        "Node Count",
						Type:        "integer",
						Description: "The number of nodes in the Aerospike cluster",
						JSONPath:    ".status.nodeCount",
					},
					{
						Name:        "Age",
						Type:        "date",
						Description: "Time elapsed since the resource was created",
						JSONPath:    ".metadata.creationTimestamp",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacebackups", aerospikev1alpha2.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group: aerospikev1alpha2.SchemeGroupVersion.Group,
				Versions: []extsv1beta1.CustomResourceDefinitionVersion{
					{
						Name:    aerospikev1alpha2.SchemeGroupVersion.Version,
						Served:  true,
						Storage: true,
					},
					{
						Name:    aerospikev1alpha1.SchemeGroupVersion.Version,
						Served:  true,
						Storage: false,
					},
				},
				Scope: extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural:     AerospikeNamespaceBackupPlural,
					Kind:       AerospikeNamespaceBackupKind,
					ShortNames: []string{AerospikeNamespaceBackupShort},
				},
				Validation: &extsv1beta1.CustomResourceValidation{
					OpenAPIV3Schema: &extsv1beta1.JSONSchemaProps{
						Properties: map[string]extsv1beta1.JSONSchemaProps{
							"spec": {
								Properties: map[string]extsv1beta1.JSONSchemaProps{
									"target":  backupRestoreTargetProps,
									"storage": backupStorageSpecProps,
									"ttl": {
										Type:    "string",
										Pattern: ttlPattern,
									},
								},
								Required: []string{
									"target",
								},
							},
						},
					},
				},
				Subresources: &extsv1beta1.CustomResourceSubresources{
					Status: &extsv1beta1.CustomResourceSubresourceStatus{},
				},
				AdditionalPrinterColumns: []extsv1beta1.CustomResourceColumnDefinition{
					{
						Name:        "Target Cluster",
						Type:        "string",
						Description: "The name of the Aerospike cluster targeted by the backup operation",
						JSONPath:    ".spec.target.cluster",
					},
					{
						Name:        "Target Namespace",
						Type:        "string",
						Description: "The name of the Aerospike namespace targeted by the backup operation",
						JSONPath:    ".spec.target.namespace",
					},
					{
						Name:        "Age",
						Type:        "date",
						Description: "Time elapsed since the resource was created",
						JSONPath:    ".metadata.creationTimestamp",
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacerestores", aerospikev1alpha2.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group: aerospikev1alpha2.SchemeGroupVersion.Group,
				Versions: []extsv1beta1.CustomResourceDefinitionVersion{
					{
						Name:    aerospikev1alpha2.SchemeGroupVersion.Version,
						Served:  true,
						Storage: true,
					},
					{
						Name:    aerospikev1alpha1.SchemeGroupVersion.Version,
						Served:  true,
						Storage: false,
					},
				},
				Scope: extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural:     AerospikeNamespaceRestorePlural,
					Kind:       AerospikeNamespaceRestoreKind,
					ShortNames: []string{AerospikeNamespaceRestoreShort},
				},
				Validation: &extsv1beta1.CustomResourceValidation{
					OpenAPIV3Schema: &extsv1beta1.JSONSchemaProps{
						Properties: map[string]extsv1beta1.JSONSchemaProps{
							"spec": {
								Properties: map[string]extsv1beta1.JSONSchemaProps{
									"target":  backupRestoreTargetProps,
									"storage": backupStorageSpecProps,
								},
								Required: []string{
									"target",
								},
							},
						},
					},
				},
				Subresources: &extsv1beta1.CustomResourceSubresources{
					Status: &extsv1beta1.CustomResourceSubresourceStatus{},
				},
				AdditionalPrinterColumns: []extsv1beta1.CustomResourceColumnDefinition{
					{
						Name:        "Target Cluster",
						Type:        "string",
						Description: "The name of the Aerospike cluster targeted by the restore operation",
						JSONPath:    ".spec.target.cluster",
					},
					{
						Name:        "Target Namespace",
						Type:        "string",
						Description: "The name of the Aerospike namespace targeted by the restore operation",
						JSONPath:    ".spec.target.namespace",
					},
					{
						Name:        "Age",
						Type:        "date",
						Description: "Time elapsed since the resource was created",
						JSONPath:    ".metadata.creationTimestamp",
					},
				},
			},
		},
	}
)
