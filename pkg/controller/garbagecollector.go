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

	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/garbagecollector"
)

const (
	// garbageCollectorControllerDefaultThreadiness is the number of workers the backup
	// controller will use to process items from the queue.
	garbageCollectorControllerDefaultThreadiness = 2
)

// AerospikeGarbageCollectorController is the controller for AerospikeNamespaceBackup resources
type AerospikeGarbageCollectorController struct {
	*genericController
	aerospikeNamespaceBackupLister   aerospikelisters.AerospikeNamespaceBackupLister
	aerospikeNamespaceBackupsHandler *garbagecollector.AerospikeNamespaceBackupHandler
}

// NewGarbageCollectorController returns a new controller for AerospikeNamespaceBackup resources
func NewGarbageCollectorController(
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *AerospikeGarbageCollectorController {

	// obtain references to shared informers for the required types
	aerospikeNamespaceBackupInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeNamespaceBackups()

	// obtain references to listers for the required types
	aerospikeNamespaceBackupLister := aerospikeNamespaceBackupInformer.Lister()

	c := &AerospikeGarbageCollectorController{
		genericController:              newGenericController("aerospikegarbagecollector", garbageCollectorControllerDefaultThreadiness, kubeClient),
		aerospikeNamespaceBackupLister: aerospikeNamespaceBackupLister,
	}
	c.hasSyncedFuncs = []cache.InformerSynced{
		aerospikeNamespaceBackupInformer.Informer().HasSynced,
	}
	c.syncHandler = c.processQueueItem

	c.aerospikeNamespaceBackupsHandler = garbagecollector.NewAerospikeNamespaceBackupHandler(kubeClient, aerospikeClient, aerospikeNamespaceBackupLister, c.recorder)
	c.logger.Debug("setting up event handlers")

	// setup an event handler for when AerospikeNamespaceBackup resources change
	aerospikeNamespaceBackupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueWithPrefix(obj, asnbPrefix)
		},
		UpdateFunc: func(_, obj interface{}) {
			c.enqueue(obj)
		},
	})
	return c
}

// processQueueItem compares the actual state with the desired, and attempts to converge the two
func (c *AerospikeGarbageCollectorController) processQueueItem(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the AerospikeNamespaceBackup resource with this namespace/name
	aerospikeNamespaceBackup, err := c.aerospikeNamespaceBackupLister.AerospikeNamespaceBackups(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("aerospikenamespacebackup '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	return c.aerospikeNamespaceBackupsHandler.Handle(aerospikeNamespaceBackup.DeepCopy())
}
