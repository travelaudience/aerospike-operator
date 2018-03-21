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
	"time"

	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	Kind   = "AerospikeCluster"
	Plural = "aerospikeclusters"

	// how long to wait for the crd to become ESTABLISHED before timing out
	watchTimeout = 15 * time.Second
)

func Ensure(extsClient extsclientset.Interface) error {
	// create a CustomResourceDefinition object representing our crd
	crd := createCRDObject()
	// create the CustomResourceDefinition in the api
	if err := createCRD(extsClient, crd); err != nil {
		return err
	}
	// watch the crd, waiting for it to be established
	return awaitCRD(extsClient, crd)
}

func createCRDObject() *extsv1beta1.CustomResourceDefinition {
	return &extsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", Plural, aerospikev1alpha1.SchemeGroupVersion.Group),
		},
		Spec: extsv1beta1.CustomResourceDefinitionSpec{
			Group:   aerospikev1alpha1.SchemeGroupVersion.Group,
			Version: aerospikev1alpha1.SchemeGroupVersion.Version,
			Scope:   extsv1beta1.NamespaceScoped,
			Names: extsv1beta1.CustomResourceDefinitionNames{
				Plural: Plural,
				Kind:   Kind,
			},
		},
	}
}

func createCRD(extsClient extsclientset.Interface, crd *extsv1beta1.CustomResourceDefinition) error {
	// attempt to register our crd
	log.WithField(logfields.Kind, Kind).Debug("registering crd")
	_, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		log.WithField(logfields.Kind, Kind).Debug("crd already registered")
	}
	return nil
}

func awaitCRD(extsClient extsclientset.Interface, crd *extsv1beta1.CustomResourceDefinition) error {
	log.WithField(logfields.Kind, Kind).Debug("waiting for crd to be established")
	w, err := extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Watch(metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name==%s", crd.Name),
	})
	if err != nil {
		return err
	}
	conditions := []watch.ConditionFunc{
		func(event watch.Event) (bool, error) {
			// grab the current crd object from the event
			obj := event.Object.(*extsv1beta1.CustomResourceDefinition)
			// search for Established in .Status.Conditions and make sure it is True
			// https://github.com/kubernetes/apiextensions-apiserver/blob/kubernetes-1.9.4/pkg/apis/apiextensions/types.go#L74
			for _, cond := range obj.Status.Conditions {
				switch cond.Type {
				case extsv1beta1.Established:
					if cond.Status == extsv1beta1.ConditionTrue {
						return true, nil
					}
				}
			}
			// otherwise return false
			return false, nil
		},
	}
	last, err := watch.Until(watchTimeout, w, conditions...)
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for crd %s", meta.Key(crd))
	}

	log.WithField(logfields.Kind, Kind).Debug("crd established")
	return nil
}
