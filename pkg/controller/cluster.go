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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	aerospikelisters "github.com/travelaudience/aerospike-operator/pkg/client/listers/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/reconciler"
)

// AerospikeClusterController is the controller for AerospikeCluster resources
type AerospikeClusterController struct {
	*genericController
	aerospikeClustersLister aerospikelisters.AerospikeClusterLister
	reconciler              *reconciler.AerospikeClusterReconciler
}

// NewAerospikeClusterController returns a new controller for AerospikeCluster resources
func NewAerospikeClusterController(
	kubeClient kubernetes.Interface,
	aerospikeClient aerospikeclientset.Interface,
	kubeInformerFactory kubeinformers.SharedInformerFactory,
	aerospikeInformerFactory aerospikeinformers.SharedInformerFactory) *AerospikeClusterController {

	// obtain references to shared informers for the required types
	podInformer := kubeInformerFactory.Core().V1().Pods()
	configMapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	pvcInformer := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	scInformer := kubeInformerFactory.Storage().V1().StorageClasses()
	aerospikeClusterInformer := aerospikeInformerFactory.Aerospike().V1alpha1().AerospikeClusters()

	// obtain references to listers for the required types
	podsLister := podInformer.Lister()
	configMapsLister := configMapInformer.Lister()
	servicesLister := serviceInformer.Lister()
	pvcsLister := pvcInformer.Lister()
	scsLister := scInformer.Lister()
	aerospikeClustersLister := aerospikeClusterInformer.Lister()

	c := &AerospikeClusterController{
		genericController:       newGenericController("aerospikecluster", kubeClient),
		aerospikeClustersLister: aerospikeClustersLister,
	}
	c.hasSyncedFuncs = []cache.InformerSynced{
		podInformer.Informer().HasSynced,
		configMapInformer.Informer().HasSynced,
		serviceInformer.Informer().HasSynced,
		pvcInformer.Informer().HasSynced,
		scInformer.Informer().HasSynced,
		aerospikeClusterInformer.Informer().HasSynced,
	}
	c.syncHandler = c.processQueueItem
	c.reconciler = reconciler.New(kubeClient, aerospikeClient, podsLister, configMapsLister, servicesLister, pvcsLister, scsLister, c.recorder)

	c.logger.Debug("setting up event handlers")

	// setup an event handler for when AerospikeCluster resources change
	aerospikeClusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.enqueue,
		UpdateFunc: func(_, obj interface{}) {
			c.enqueue(obj)
		},
	})
	// setup an event handler for when Pod resources change. This
	// handler will lookup the owner of the given Pod, and if it is
	// owned by a AerospikeCluster resource will enqueue that AerospikeCluster resource for
	// processing. This way, we don't need to implement custom logic for
	// handling Pod resources. More info on this pattern:
	// https://github.com/kubernetes/community/blob/8cafef897a22026d42f5e5bb3f104febe7e29830/contributors/devel/controllers.md
	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: c.handleObject,
		UpdateFunc: func(old, new interface{}) {
			newPod := new.(*corev1.Pod)
			oldPod := old.(*corev1.Pod)
			if newPod.ResourceVersion == oldPod.ResourceVersion {
				// Periodic resync will send update events for all known Pods.
				// Two different versions of the same Pod will always have different RVs.
				return
			}
			c.handleObject(new)
		},
		DeleteFunc: c.handleObject,
	})

	return c
}

// processQueueItem compares the actual state with the desired, and attempts to converge the two
func (c *AerospikeClusterController) processQueueItem(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the AerospikeCluster resource with this namespace/name
	aerospikeCluster, err := c.aerospikeClustersLister.AerospikeClusters(namespace).Get(name)
	if err != nil {
		// The AerospikeCluster resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("aerospikecluster '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	// deepcopy aerospikeCluster before reconciling so we don't possibly mutate the cache
	err = c.reconciler.MaybeReconcile(aerospikeCluster.DeepCopy())
	if err != nil {
		return err
	}
	return nil
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the AerospikeCluster resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that AerospikeCluster resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *AerospikeClusterController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		c.logger.Debugf("recovered deleted object '%s' from tombstone", object.GetName())
	}
	c.logger.Debugf("processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		// If this object is not owned by a AerospikeCluster, we should not do anything more
		// with it.
		if ownerRef.Kind != "AerospikeCluster" {
			return
		}

		aerospikeCluster, err := c.aerospikeClustersLister.AerospikeClusters(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			c.logger.Debugf("ignoring orphaned object '%s' of aerospikeCluster '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		c.enqueue(aerospikeCluster)
		return
	}
}
