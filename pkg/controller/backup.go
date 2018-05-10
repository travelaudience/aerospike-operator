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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/travelaudience/aerospike-operator/pkg/backuphandler"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
)

// AerospikeNamespaceBackupController is the controller for AerospikeNamespaceBackup resources
type AerospikeNamespaceBackupController struct {
	*genericController
	aerospikeNamespaceBackupLister aerospikelisters.AerospikeNamespaceBackupLister
	handler                        *backuphandler.AerospikeBackupsHandler
}

// NewAerospikeNamespaceBackupController returns a new controller for AerospikeNamespaceBackup resources
func NewAerospikeNamespaceBackupController(
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *AerospikeNamespaceBackupController {

	// obtain references to shared informers for the required types
	jobInformer := kubeInformerFactory.Batch().V1().Jobs()
	secretInformer := kubeInformerFactory.Core().V1().Secrets()
	aerospikeClusterInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeClusters()
	aerospikeNamespaceBackupInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeNamespaceBackups()

	// obtain references to listers for the required types
	jobsLister := jobInformer.Lister()
	secretsLister := secretInformer.Lister()
	aerospikeClustersLister := aerospikeClusterInformer.Lister()
	aerospikeNamespaceBackupLister := aerospikeNamespaceBackupInformer.Lister()

	c := &AerospikeNamespaceBackupController{
		genericController:              newGenericController("aerospikenamespacebackup", kubeClient),
		aerospikeNamespaceBackupLister: aerospikeNamespaceBackupLister,
	}
	c.hasSyncedFuncs = []cache.InformerSynced{
		aerospikeNamespaceBackupInformer.Informer().HasSynced,
	}
	c.syncHandler = c.processQueueItem

	c.handler = backuphandler.New(kubeClient, aerospikeClient, aerospikeClustersLister, jobsLister, secretsLister, c.recorder)
	c.logger.Debug("setting up event handlers")

	// setup an event handler for when AerospikeNamespaceBackup resources change
	aerospikeNamespaceBackupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, obj interface{}) {
			c.enqueue(obj)
		},
	})

	return c
}

// processQueueItem compares the actual state with the desired, and attempts to converge the two
func (c *AerospikeNamespaceBackupController) processQueueItem(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the AerospikeNamespaceBackup resource with this namespace/name
	aerospikeNamespaceBackup, err := c.aerospikeNamespaceBackupLister.AerospikeNamespaceBackups(namespace).Get(name)
	if err != nil {
		// The AerospikeNamespaceBackup resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("aerospikenamespacebackup '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// deepcopy aerospikeNamespaceRestore before handle it so we don't possibly mutate the cache
	err = c.handler.Handle(aerospikeNamespaceBackup.DeepCopy())
	if err != nil {
		return err
	}
	return nil
}
