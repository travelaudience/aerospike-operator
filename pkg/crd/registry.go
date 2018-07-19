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
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
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
		if err := r.awaitCRD(crd, watchTimeout); err != nil {
			return err
		}
	}
	return nil
}

func (r *CRDRegistry) createCRD(crd *extsv1beta1.CustomResourceDefinition) error {
	// attempt to register the crd as instructed
	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("registering crd")
	_, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err == nil {
		// registration was successful
		return nil
	}
	if !errors.IsAlreadyExists(err) {
		// the crd doesn't exist yet but we got an unexpected error while creating
		return err
	}

	// at this point the crd already exists but its spec may differ since the
	// api is not stable yet. as such, we must do our best to update the crd.

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("crd is already registered")

	// fetch the latest version of the crd
	d, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
	if err != nil {
		// we've failed to fetch the latest version of the crd
		return nil
	}
	if reflect.DeepEqual(d.Spec, crd.Spec) {
		// if the specs match there's nothing to do
		return nil
	}

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("updating crd")

	// set the resulting object's spec according to the current spec
	d.Spec = crd.Spec

	// attempt to update the crd
	if _, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Update(d); err != nil {
		return err
	}

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("crd has been updated")

	return nil
}

func (r *CRDRegistry) awaitCRD(crd *extsv1beta1.CustomResourceDefinition, timeout time.Duration) error {
	start := time.Now()
	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("waiting for crd to be established")
	w, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Watch(listoptions.ObjectByNameAndVersion(crd.Name, crd.ResourceVersion))
	if err != nil {
		return err
	}

	lastCRD := crd
	last, err := watch.Until(timeout, w, func(event watch.Event) (bool, error) {
		// grab the current crd object from the event
		obj := event.Object.(*extsv1beta1.CustomResourceDefinition)
		// search for Established in .Status.Conditions and make sure it is True
		// https://github.com/kubernetes/apiextensions-apiserver/blob/kubernetes-1.10.5/pkg/apis/apiextensions/types.go#L81
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
		// ErrWatchClosed is returned when the watch channel is closed before timeout in Until
		if err == watch.ErrWatchClosed {
			// re-establish retry until we reach the timeout
			if t := timeout - time.Since(start); t > 0 {
				// use the resource object of the last event if it exists
				if last != nil {
					lastCRD = last.Object.(*extsv1beta1.CustomResourceDefinition)
				}
				return r.awaitCRD(lastCRD, t)
			}
		}
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for crd %s", meta.Key(crd))
	}

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Info("crd established")
	return nil
}
