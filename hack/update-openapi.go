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

package main

import (
	"fmt"

	"github.com/appscode/kutil/openapi"
	"github.com/go-openapi/spec"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/kube-openapi/pkg/common"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
)

func main() {
	s := runtime.NewScheme()
	c := serializer.NewCodecFactory(s)

	utilruntime.Must(aerospikev1alpha1.AddToScheme(s))
	utilruntime.Must(s.SetVersionPriority(aerospikev1alpha1.SchemeGroupVersion))

	cfg := openapi.Config{
		Scheme: s,
		Codecs: c,
		Info: spec.InfoProps{
			Description: "aerospike-operator manages Aerospike clusters atop Kubernetes, automating their creation and administration.",
			Title:       aerospikev1alpha1.SchemeGroupVersion.Group,
			Version:     aerospikev1alpha1.SchemeGroupVersion.Version,
			License: &spec.License{
				Name: "Apache 2.0",
				URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
			},
		},
		OpenAPIDefinitions: []common.GetOpenAPIDefinitions{
			aerospikev1alpha1.GetOpenAPIDefinitions,
		},
		Resources: []openapi.TypeInfo{
			{
				GroupVersion:    aerospikev1alpha1.SchemeGroupVersion,
				Resource:        crd.AerospikeClusterPlural,
				Kind:            crd.AerospikeClusterKind,
				NamespaceScoped: true,
			},
			{
				GroupVersion:    aerospikev1alpha1.SchemeGroupVersion,
				Resource:        crd.AerospikeNamespaceBackupPlural,
				Kind:            crd.AerospikeNamespaceBackupKind,
				NamespaceScoped: true,
			},
			{
				GroupVersion:    aerospikev1alpha1.SchemeGroupVersion,
				Resource:        crd.AerospikeNamespaceRestorePlural,
				Kind:            crd.AerospikeNamespaceRestoreKind,
				NamespaceScoped: true,
			},
		},
	}
	if out, err := openapi.RenderOpenAPISpec(cfg); err != nil {
		log.Fatalf("failed to generate spec: %v", err)
	} else {
		fmt.Println(out)
	}
}
