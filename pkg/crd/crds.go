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
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacesbackups", aerospikev1alpha1.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
				Version: aerospikev1alpha1.SchemeGroupVersion.Version,
				Scope:   extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural: "aerospikenamespacesbackups",
					Kind:   "AerospikeNamespaceBackup",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s.%s", "aerospikenamespacesrestores", aerospikev1alpha1.SchemeGroupVersion.Group),
			},
			Spec: extsv1beta1.CustomResourceDefinitionSpec{
				Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
				Version: aerospikev1alpha1.SchemeGroupVersion.Version,
				Scope:   extsv1beta1.NamespaceScoped,
				Names: extsv1beta1.CustomResourceDefinitionNames{
					Plural: "aerospikenamespacesrestores",
					Kind:   "AerospikeNamespaceRestore",
				},
			},
		},
	}
)
