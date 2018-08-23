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
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
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
	// asnbPrefix is the prefix used when enqueuing AerospikeNamespaceBackup candidates for garbage collection
	asnbPrefix = "asnb"
	// pvcPrefix is the prefix used when enqueuing PersistentVolumeClaims candidates for garbage collection
	pvcPrefix = "pvc"
)

// AerospikeGarbageCollectorController is the controller for AerospikeNamespaceBackup resources
type AerospikeGarbageCollectorController struct {
	*genericController
	aerospikeNamespaceBackupLister   aerospikelisters.AerospikeNamespaceBackupLister
	aerospikeNamespaceBackupsHandler *garbagecollector.AerospikeNamespaceBackupHandler
	pvcsLister                       listersv1.PersistentVolumeClaimLister
	pvcsHandler                      *garbagecollector.PVCsHandler
}

// NewGarbageCollectorController returns a new controller for AerospikeNamespaceBackup resources
func NewGarbageCollectorController(
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface,
	kubeInformerFactory informers.SharedInformerFactory,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *AerospikeGarbageCollectorController {

	// obtain references to shared informers for the required types
	aerospikeNamespaceBackupInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeNamespaceBackups()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()

	// obtain references to listers for the required types
	aerospikeNamespaceBackupLister := aerospikeNamespaceBackupInformer.Lister()
	pvcsLister := pvcInformer.Lister()

	c := &AerospikeGarbageCollectorController{
		genericController:              newGenericController("aerospikegarbagecollector", garbageCollectorControllerDefaultThreadiness, kubeClient),
		aerospikeNamespaceBackupLister: aerospikeNamespaceBackupLister,
		pvcsLister:                     pvcsLister,
	}
	c.hasSyncedFuncs = []cache.InformerSynced{
		aerospikeNamespaceBackupInformer.Informer().HasSynced,
		pvcInformer.Informer().HasSynced,
	}
	c.syncHandler = c.processQueueItem

	c.aerospikeNamespaceBackupsHandler = garbagecollector.NewAerospikeNamespaceBackupHandler(kubeClient, aerospikeClient, aerospikeNamespaceBackupLister, c.recorder)
	c.pvcsHandler = garbagecollector.NewPVCsGCHandler(kubeClient, pvcsLister, c.recorder)

	c.logger.Debug("setting up event handlers")

	// setup an event handler for when AerospikeNamespaceBackup resources change
	aerospikeNamespaceBackupInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueWithPrefix(obj, asnbPrefix)
		},
		UpdateFunc: func(_, obj interface{}) {
			c.enqueueWithPrefix(obj, asnbPrefix)
		},
	})

	// setup an event handler for when PersistentVolumeClaims resources change
	pvcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(_, obj interface{}) {
			c.handleObject(obj)
		},
	})

	return c
}

// processQueueItem compares the actual state with the desired, and attempts to converge the two
func (c *AerospikeGarbageCollectorController) processQueueItem(prefixedKey string) error {
	ss := strings.Split(prefixedKey, ":")
	if len(ss) != 2 {
		return fmt.Errorf("invalid key format for garbagecollector controller: %q", prefixedKey)
	}
	prefix, key := ss[0], ss[1]

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	switch prefix {
	case asnbPrefix:
		// Get the AerospikeNamespaceBackup resource with this namespace/name
		aerospikeNamespaceBackup, err := c.aerospikeNamespaceBackupLister.AerospikeNamespaceBackups(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				runtime.HandleError(fmt.Errorf("aerospikenamespacebackup '%s' in work queue no longer exists", prefixedKey))
				return nil
			}
			return err
		}
		return c.aerospikeNamespaceBackupsHandler.Handle(aerospikeNamespaceBackup.DeepCopy())
	case pvcPrefix:
		// Get the PersistentVolumeClaim resource with this namespace/name
		pvc, err := c.pvcsLister.PersistentVolumeClaims(namespace).Get(name)
		if err != nil {
			if errors.IsNotFound(err) {
				runtime.HandleError(fmt.Errorf("persistentvolumeclaim '%s' in work queue no longer exists", prefixedKey))
				return nil
			}
			return err
		}
		return c.pvcsHandler.Handle(pvc.DeepCopy())
	default:
		return fmt.Errorf("invalid prefix %q", prefix)
	}
}

// handleObject will take any resource implementing metav1.Object and enqueue
// it only if it is owned by an AerospikeCluster.
func (c *AerospikeGarbageCollectorController) handleObject(obj interface{}) {
	if object, ok := obj.(metav1.Object); ok {
		c.logger.Debugf("processing object: %s", object.GetName())
		// If this object is owned by an AerospikeCluster, we enqueue it.
		if ownerRef := metav1.GetControllerOf(object); ownerRef != nil && ownerRef.Kind == "AerospikeCluster" {
			c.enqueueWithPrefix(obj, pvcPrefix)
		}
	}
}

// enqueueWithPrefix takes a resource and and a prefix, and converts it into a
// prefix:namespace/name string which is then put onto the work queue.
func (c *genericController) enqueueWithPrefix(obj interface{}, prefix string) {
	if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
		c.workqueue.AddRateLimited(fmt.Sprintf("%s:%s", prefix, key))
	} else {
		runtime.HandleError(err)
	}
}
