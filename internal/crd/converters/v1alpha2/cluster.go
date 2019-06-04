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

	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/common"
	"github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/internal/crd"
	"github.com/travelaudience/aerospike-operator/internal/meta"
)

var (
	// defaultSecretKey duplicates the DefaultSecretFilename constant so we can later take its address.
	defaultSecreyKey = common.DefaultSecretFilename
)

func convertAerospikeClusters(extsClient *extsclientset.Clientset, aerospikeClient *aerospikeclientset.Clientset) error {
	// fetch the aerospikecluster crd so we can understand if v1alpha1 is still being used as a storage version
	asccrd, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.AerospikeClusterCRDName, v1.GetOptions{})
	if err != nil {
		return nil
	}
	// idx will hold the index of "v1alpha1" in the slice of stored versions, or -1 if not present
	idx := -1
	for i, v := range asccrd.Status.StoredVersions {
		if v == v1alpha1.SchemeGroupVersion.Version {
			idx = i
		}
	}
	// if v1alpha1 is not found, return immediately
	if idx < 0 {
		log.Debug("no aerospikecluster resources require conversion")
		return nil
	}

	// list all existing v1alpha1 aerospikecluster resources across all namespaces
	ascs, err := aerospikeClient.AerospikeV1alpha1().AerospikeClusters(v1.NamespaceAll).List(v1.ListOptions{})
	if err != nil {
		return nil
	}
	// apply conversion to every listed aerospikecluster resource
	for _, asc := range ascs.Items {
		// initialize .metadata.annotations if required
		if asc.ObjectMeta.Annotations == nil {
			asc.ObjectMeta.Annotations = make(map[string]string)
		}
		// if the "a.t.c/converted-from" annotation is set on the resource, there is nothing to do.
		if _, ok := asc.ObjectMeta.Annotations[convertedFromAnnotationKey]; ok {
			log.Debugf("aerospikecluster %s does not require conversion", meta.Key(asc))
			continue
		}
		log.Debugf("aerospikecluster %s requires conversion", meta.Key(asc))
		// set the "a.t.c/converted-from" annotation on the resource before updating
		asc.ObjectMeta.Annotations[convertedFromAnnotationKey] = v1alpha1.SchemeGroupVersion.Version
		// perform an update on the unchanged aerospikecluster resource so it is upgraded to the current storage version
		// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version
		if _, err := aerospikeClient.AerospikeV1alpha1().AerospikeClusters(asc.Namespace).Update(&asc); err != nil {
			return err
		}
	}

	// remove "v1alpha1" from the slice of stored versions
	asccrd.Status.StoredVersions = append(asccrd.Status.StoredVersions[:idx], asccrd.Status.StoredVersions[idx+1:]...)
	// update the aerospikecluster crd
	if _, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().UpdateStatus(asccrd); err != nil {
		return err
	}

	return nil
}
