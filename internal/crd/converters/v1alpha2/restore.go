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
	log "github.com/sirupsen/logrus"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/internal/crd"
	"github.com/travelaudience/aerospike-operator/internal/meta"
)

func convertAerospikeNamespaceRestores(extsClient *extsclientset.Clientset, aerospikeClient *aerospikeclientset.Clientset) error {
	// fetch the aerospikenamespacerestore crd so we can understand if v1alpha1 is still being used as a storage version
	asnrcrd, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.AerospikeNamespaceRestoreCRDName, v1.GetOptions{})
	if err != nil {
		return nil
	}
	// idx will hold the index of "v1alpha1" in the slice of stored versions, or -1 if not present
	idx := -1
	for i, v := range asnrcrd.Status.StoredVersions {
		if v == v1alpha1.SchemeGroupVersion.Version {
			idx = i
		}
	}
	// if v1alpha1 is not found, return immediately
	if idx < 0 {
		log.Debug("no aerospikenamespacerestore resources require conversion")
		return nil
	}

	// list all existing v1alpha1 aerospikenamespacerestore resources across all namespaces
	asnr, err := aerospikeClient.AerospikeV1alpha1().AerospikeNamespaceRestores(v1.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		return nil
	}
	// apply conversion to every listed aerospikenamespacerestore resource
	for _, asnr := range asnr.Items {
		// initialize .metadata.annotations if required
		if asnr.ObjectMeta.Annotations == nil {
			asnr.ObjectMeta.Annotations = make(map[string]string)
		}
		// if the "a.t.c/converted-from" annotation is set on the resource, there is nothing to do.
		if _, ok := asnr.ObjectMeta.Annotations[convertedFromAnnotationKey]; ok {
			log.Debugf("aerospikenamespacerestore %s does not require conversion", meta.Key(asnr))
			continue
		}
		log.Debugf("aerospikenamespacerestore %s requires conversion", meta.Key(asnr))
		// set the "a.t.c/converted-from" annotation on the resource before updating
		asnr.ObjectMeta.Annotations[convertedFromAnnotationKey] = v1alpha1.SchemeGroupVersion.Version
		// perform an update on the unchanged aerospikenamespacerestore resource so it is upgraded to the current storage version
		// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version
		if _, err := aerospikeClient.AerospikeV1alpha1().AerospikeNamespaceRestores(asnr.Namespace).Update(&asnr); err != nil {
			return err
		}
	}

	// remove "v1alpha1" from the slice of stored versions
	asnrcrd.Status.StoredVersions = append(asnrcrd.Status.StoredVersions[:idx], asnrcrd.Status.StoredVersions[idx+1:]...)
	// update the aerospikenamespacerestore crd
	if _, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().UpdateStatus(asnrcrd); err != nil {
		return err
	}

	return nil
}
