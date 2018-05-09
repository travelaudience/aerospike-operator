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

package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
)

// AerospikeNamespaceRestoreController is the controller for AerospikeNamespaceRestore resources
type AerospikeNamespaceRestoreController struct {
	*genericController
	aerospikeNamespaceRestoreLister aerospikelisters.AerospikeNamespaceRestoreLister
}

// NewAerospikeNamespaceRestoreController returns a new controller for AerospikeNamespaceRestore objects
func NewAerospikeNamespaceRestoreController(
	kubeClient kubernetes.Interface,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *AerospikeNamespaceRestoreController {

	// obtain references to shared informers for the required types
	aerospikeNamespaceRestoreInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeNamespaceRestores()

	// obtain references to listers for the required types
	aerospikeNamespaceRestoreLister := aerospikeNamespaceRestoreInformer.Lister()

	c := &AerospikeNamespaceRestoreController{
		genericController:               newGenericController("aerospikenamespacerestore", kubeClient),
		aerospikeNamespaceRestoreLister: aerospikeNamespaceRestoreLister,
	}
	c.hasSyncedFuncs = []cache.InformerSynced{
		aerospikeNamespaceRestoreInformer.Informer().HasSynced,
	}
	c.syncHandler = c.processQueueItem

	c.logger.Debug("setting up event handlers")

	// setup an event handler for when AerospikeNamespaceRestore resources change
	aerospikeNamespaceRestoreInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, obj interface{}) {
			c.enqueue(obj)
		},
	})

	return c
}

// processQueueItem compares the actual state with the desired, and attempts to converge the two
func (c *AerospikeNamespaceRestoreController) processQueueItem(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the AerospikeNamespaceRestore resource with this namespace/name
	aerospikeNamespaceRestore, err := c.aerospikeNamespaceRestoreLister.AerospikeNamespaceRestores(namespace).Get(name)
	if err != nil {
		// The AerospikeNamespaceRestores resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("aerospikenamespacerestore '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	c.logger.Debugf("should process %s", aerospikeNamespaceRestore.UID) // TODO implement
	return nil
}
