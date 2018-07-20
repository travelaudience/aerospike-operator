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
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/kube-openapi/pkg/common"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike"
	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
)

func main() {
	f := make(announced.APIGroupFactoryRegistry)
	r := registered.NewOrDie("")
	s := runtime.NewScheme()
	c := serializer.NewCodecFactory(s)
	g := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:              aerospike.GroupName,
			RootScopedKinds:        sets.NewString(),
			VersionPreferenceOrder: []string{aerospikev1alpha1.SchemeGroupVersion.Version},
		},
		announced.VersionToSchemeFunc{
			aerospikev1alpha1.SchemeGroupVersion.Version: aerospikev1alpha1.AddToScheme,
		},
	)
	if err := g.Announce(f).RegisterAndEnable(r, s); err != nil {
		log.Fatalf("failed to generate spec: %v", err)
	}

	cfg := openapi.Config{
		Registry: r,
		Scheme:   s,
		Codecs:   c,
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
		Resources: []schema.GroupVersionResource{
			aerospikev1alpha1.SchemeGroupVersion.WithResource(crd.AerospikeClusterPlural),
			aerospikev1alpha1.SchemeGroupVersion.WithResource(crd.AerospikeNamespaceBackupPlural),
			aerospikev1alpha1.SchemeGroupVersion.WithResource(crd.AerospikeNamespaceRestorePlural),
		},
	}
	if out, err := openapi.RenderOpenAPISpec(cfg); err != nil {
		log.Fatalf("failed to generate spec: %v", err)
	} else {
		fmt.Println(out)
	}
}
