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
	"context"

	log "github.com/sirupsen/logrus"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

func convertAerospikeNamespaceBackups(extsClient *extsclientset.Clientset, aerospikeClient *aerospikeclientset.Clientset) error {
	// fetch the aerospikenamespacebackup crd so we can understand if v1alpha1 is still being used as a storage version
	asnbcrd, err := extsClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crd.AerospikeNamespaceBackupCRDName, v1.GetOptions{})
	if err != nil {
		return nil
	}
	// idx will hold the index of "v1alpha1" in the slice of stored versions, or -1 if not present
	idx := -1
	for i, v := range asnbcrd.Status.StoredVersions {
		if v == v1alpha1.SchemeGroupVersion.Version {
			idx = i
		}
	}
	// if v1alpha1 is not found, return immediately
	if idx < 0 {
		log.Debug("no aerospikenamespacebackup resources require conversion")
		return nil
	}

	// list all existing v1alpha1 aerospikenamespacebackup resources across all namespaces
	asnbs, err := aerospikeClient.AerospikeV1alpha1().AerospikeNamespaceBackups(v1.NamespaceAll).List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil
	}
	// apply conversion to every listed aerospikenamespacebackup resource
	for _, asnb := range asnbs.Items {
		// initialize .metadata.annotations if required
		if asnb.ObjectMeta.Annotations == nil {
			asnb.ObjectMeta.Annotations = make(map[string]string)
		}
		// if the "a.t.c/converted-from" annotation is set on the resource, there is nothing to do.
		if _, ok := asnb.ObjectMeta.Annotations[convertedFromAnnotationKey]; ok {
			log.Debugf("aerospikenamespacebackup %s does not require conversion", meta.Key(asnb))
			continue
		}
		log.Debugf("aerospikenamespacebackup %s requires conversion", meta.Key(asnb))
		// set the "a.t.c/converted-from" annotation on the resource before updating
		asnb.ObjectMeta.Annotations[convertedFromAnnotationKey] = v1alpha1.SchemeGroupVersion.Version
		// perform an update on the unchanged aerospikenamespacebackup resource so it is upgraded to the current storage version
		// https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definition-versioning/#upgrade-existing-objects-to-a-new-stored-version
		if _, err := aerospikeClient.AerospikeV1alpha1().AerospikeNamespaceBackups(asnb.Namespace).Update(context.TODO(), &asnb, v1.UpdateOptions{}); err != nil {
			return err
		}
	}

	// remove "v1alpha1" from the slice of stored versions
	asnbcrd.Status.StoredVersions = append(asnbcrd.Status.StoredVersions[:idx], asnbcrd.Status.StoredVersions[idx+1:]...)
	// update the aerospikenamespacebackup crd
	if _, err := extsClient.ApiextensionsV1().CustomResourceDefinitions().UpdateStatus(context.TODO(), asnbcrd, v1.UpdateOptions{}); err != nil {
		return err
	}

	return nil
}
