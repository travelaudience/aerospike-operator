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
	"context"
	"fmt"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	extsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/internal/client/clientset/versioned"
	"github.com/travelaudience/aerospike-operator/internal/logfields"
	"github.com/travelaudience/aerospike-operator/internal/utils/selectors"
)

const (
	// how long to wait for every crd to become ESTABLISHED before timing out
	watchTimeout = 15 * time.Second
)

// CRDRegistry registers our CRDs, waiting for them to be established.
type CRDRegistry struct {
	extsClient      extsclientset.Interface
	aerospikeClient aerospikeclientset.Interface
}

// NewCRDRegistry creates a new CRDRegistry using the given clientset.
func NewCRDRegistry(extsClient extsclientset.Interface, aerospikeClient aerospikeclientset.Interface) *CRDRegistry {
	return &CRDRegistry{
		extsClient:      extsClient,
		aerospikeClient: aerospikeClient,
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
	d, err := r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
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

func isCRDEstablished(crd *extsv1beta1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		if cond.Type == extsv1beta1.Established {
			if cond.Status == extsv1beta1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func (r *CRDRegistry) awaitCRD(crd *extsv1beta1.CustomResourceDefinition, timeout time.Duration) error {
	// Grab a ListerWatcher with which we can watch the CustomResourceDefinition resource.
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = selectors.ObjectByCoordinates(crd.Namespace, crd.Name).String()
			return r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watchapi.Interface, error) {
			options.FieldSelector = selectors.ObjectByCoordinates(crd.Namespace, crd.Name).String()
			return r.extsClient.ApiextensionsV1beta1().CustomResourceDefinitions().Watch(options)
		},
	}

	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("waiting for crd to become established")

	// Watch for updates to the specified CustomResourceDefinition resource until it reaches the "Established" condition or until "waitCRDReadyTimeout" elapses.
	ctx, fn := context.WithTimeout(context.Background(), watchTimeout)
	defer fn()
	last, err := watch.UntilWithSync(ctx, lw, &extsv1beta1.CustomResourceDefinition{}, nil, func(event watchapi.Event) (bool, error) {
		// Grab the current resource from the event.
		obj := event.Object.(*extsv1beta1.CustomResourceDefinition)
		// Return true if and only if the CustomResourceDefinition resource has reached the "Established" condition.
		return isCRDEstablished(obj), nil
	})
	if err != nil {
		// We've got an error while watching the specified CustomResourceDefinition resource.
		return err
	}
	if last == nil {
		// We've got no events for the CustomResourceDefinition resource, which represents an error.
		return fmt.Errorf("no events received for crd %q", crd.Name)
	}

	// At this point we are sure the CustomResourceDefinition resource has reached the "Established" condition, so we return.
	log.WithField(logfields.Kind, crd.Spec.Names.Kind).Debug("crd established")
	return nil
}
