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

	log "github.com/sirupsen/logrus"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
)

const (
	// how long to wait for every crd to become ESTABLISHED before timing out
	watchTimeout = 15 * time.Second
)

// CRDRegistry registers our CRDs, waiting for them to be established.
type CRDRegistry struct {
	extsClient extsclientset.Interface
}

// NewCRDRegistry creates a new CRDRegistry using the given clientset.
func NewCRDRegistry(extsClient extsclientset.Interface) *CRDRegistry {
	return &CRDRegistry{
		extsClient: extsClient,
	}
}

// RegisterCRDs registers our CRDs, waiting for them to be established.
func (r *CRDRegistry) RegisterCRDs() error {
	for _, crd := range crds {
		// create the CustomResourceDefinition in the api
		if err := r.createCRD(crd); err != nil {
			return err
		}
		// wait for the CustomResourceDefinition to be established
		if err := r.awaitCRD(crd); err != nil {
			return err
		}
	}
	return nil
}

func (r *CRDRegistry) createCRD(crd *extsv1beta1.CustomResourceDefinition) error {
	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("registering crd")
	_, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("crd already registered")
	}
	return nil
}

func (r *CRDRegistry) awaitCRD(crd *extsv1beta1.CustomResourceDefinition) error {
	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("waiting for crd to be established")
	w, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Watch(selectors.ObjectByName(crd.Name))
	if err != nil {
		return err
	}
	last, err := watch.Until(watchTimeout, w, func(event watch.Event) (bool, error) {
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
	})
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for crd %s", meta.Key(crd))
	}

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("crd established")
	return nil
}
