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

package garbagecollector

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/record"

	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/reconciler"
	astime "github.com/travelaudience/aerospike-operator/pkg/utils/time"
)

type PVCsHandler struct {
	kubeclientset kubernetes.Interface
	pvcsLister    listersv1.PersistentVolumeClaimLister
	recorder      record.EventRecorder
}

func NewPVCsGCHandler(kubeclientset kubernetes.Interface,
	pvcsLister listersv1.PersistentVolumeClaimLister,
	recorder record.EventRecorder) *PVCsHandler {
	return &PVCsHandler{
		kubeclientset: kubeclientset,
		pvcsLister:    pvcsLister,
		recorder:      recorder,
	}
}

func (h *PVCsHandler) Handle(pvc *v1.PersistentVolumeClaim) error {
	log.WithFields(log.Fields{
		logfields.Key: meta.Key(pvc),
	}).Debug("checking whether pvc has expired")

	// get the timestamp at which the pvc was last unmounted.
	// if the timestamp is not available, then the pvc was not
	// unmounted, and we return immediately
	s, ok := pvc.Annotations[reconciler.LastUnmountedOnAnnotation]
	if !ok {
		return nil
	}
	lastUnmountedOn, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}

	// get the ttl from PVC annotations, which is set when the PVC is created
	ttl, ok := pvc.Annotations[reconciler.PVCTTLAnnotation]
	if !ok {
		return fmt.Errorf("could not retrieve ttl from annotations")
	}
	// get the pvc object expiration as a duration object
	objExpiration, err := astime.ParseDuration(ttl)
	if err != nil {
		return err
	}
	// check if the pvc object expiration
	// has no duration, in which case we return immediately
	if objExpiration == time.Second*0 {
		log.WithFields(log.Fields{
			logfields.Key: meta.Key(pvc),
		}).Debug("no expiration set for pvc")
		return nil
	}

	// return if the expiration has not been reached yet
	if time.Now().Before(lastUnmountedOn.Add(objExpiration)) {
		return nil
	}

	// make sure the pvc is not mounted in the its corresponding
	// pod
	if podName, ok := pvc.Annotations[reconciler.PodAnnotation]; !ok {
		return fmt.Errorf("could not retrieve pod-name from annotations")
	} else {
		if pod, err := h.kubeclientset.CoreV1().Pods(pvc.Namespace).Get(podName, metav1.GetOptions{}); err == nil {
			for _, volume := range pod.Spec.Volumes {
				if claim := volume.PersistentVolumeClaim; claim != nil {
					if claim.ClaimName == pvc.Name {
						log.WithFields(log.Fields{
							logfields.Key: meta.Key(pvc),
						}).Debugf("pvc is currently mounted in pod %q", meta.Key(pod))
						return nil
					}
				}
			}
		} else {
			if !errors.IsNotFound(err) {
				return err
			}
		}

	}

	// delete pvc resource
	if err := h.kubeclientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(pvc.Name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		logfields.Key: meta.Key(pvc),
	}).Info("expired pvc deleted by garbage collector")

	return nil
}
