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

package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	"github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/common"
	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	astime "github.com/travelaudience/aerospike-operator/pkg/utils/time"
)

var (
	volumeModeMap = map[string]v1.PersistentVolumeMode{
		common.StorageTypeDevice: v1.PersistentVolumeBlock,
		common.StorageTypeFile:   v1.PersistentVolumeFilesystem,
	}
)

type fromMostRecent []*v1.PersistentVolumeClaim

func (pvcs fromMostRecent) Len() int {
	return len(pvcs)
}

func (pvcs fromMostRecent) Swap(i, j int) {
	pvcs[i], pvcs[j] = pvcs[j], pvcs[i]
}

func (pvcs fromMostRecent) Less(i, j int) bool {
	return pvcs[j].CreationTimestamp.Before(&pvcs[i].CreationTimestamp)
}

func (r *AerospikeClusterReconciler) getPersistentVolumeClaim(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, pod *v1.Pod) (*v1.PersistentVolumeClaim, error) {
	// get all the pvcs owned by the aerospikecluster
	pvcs, err := r.pvcsLister.PersistentVolumeClaims(aerospikeCluster.Namespace).List(selectors.ResourcesByClusterName(aerospikeCluster.Name))
	if err != nil {
		return nil, err
	}
	if len(pvcs) == 0 {
		return nil, nil
	}

	// filter the ones associated with the pod
	var podPVCs []*v1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		// skip pvc if it does not belong to the right pod
		podName, ok := pvc.Annotations[PodAnnotation]
		if !ok || podName != pod.Name {
			continue
		}
		// retrieve the timestamp of when the pvc was last unmounted.
		// if not available, skip this pvc.
		lastUnmountedString, ok := pvc.Annotations[LastUnmountedOnAnnotation]
		if !ok {
			continue
		}
		lastUnmountedOn, err := time.Parse(time.RFC3339, lastUnmountedString)
		if err != nil {
			return nil, err
		}
		// retrieve the ttl from pvc to check if it has expired
		ttl, ok := pvc.Annotations[PVCTTLAnnotation]
		if !ok {
			continue
		}
		objExpiration, err := astime.ParseDuration(ttl)
		if err != nil {
			return nil, err
		}
		// if the pvc has not expired, add the pvc to the list of available pvcs
		if objExpiration == time.Second*0 || time.Now().Before(lastUnmountedOn.Add(objExpiration)) {
			podPVCs = append(podPVCs, pvc)
		}
	}
	if len(podPVCs) == 0 {
		return nil, nil
	}

	// sort the PVCs from the most recent to the oldest
	sort.Sort(fromMostRecent(podPVCs))

	log.WithFields(log.Fields{
		logfields.AerospikeCluster:      meta.Key(aerospikeCluster),
		logfields.Pod:                   meta.Key(pod),
		logfields.PersistentVolumeClaim: podPVCs[0].Name,
	}).Debug("using existing persistentvolumeclaim")
	// return the most recent pvc
	return podPVCs[0], nil
}

func (r *AerospikeClusterReconciler) createPersistentVolumeClaim(aerospikeCluster *aerospikev1alpha2.AerospikeCluster, pod *v1.Pod, namespace *aerospikev1alpha2.AerospikeNamespaceSpec) (*v1.PersistentVolumeClaim, error) {
	storageSize, err := resource.ParseQuantity(namespace.Storage.Size)
	if err != nil {
		return nil, err
	}

	volumeMode := volumeModeMap[namespace.Storage.Type]
	// get the persistentVolumeClaimTTL to be added to
	// the pvc as an annotation
	persistentVolumeClaimTTL := defaultPersistentVolumeClaimTTL
	if namespace.Storage.PersistentVolumeClaimTTL != nil {
		persistentVolumeClaimTTL = *namespace.Storage.PersistentVolumeClaimTTL
	}

	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-%s-", pod.Name, namespace.Name),
			Labels: map[string]string{
				selectors.LabelAppKey:       selectors.LabelAppVal,
				selectors.LabelNamespaceKey: namespace.Name,
				selectors.LabelClusterKey:   aerospikeCluster.Name,
			},
			Namespace: aerospikeCluster.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         aerospikev1alpha2.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeClusterKind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
			Annotations: map[string]string{
				PodAnnotation:    pod.Name,
				PVCTTLAnnotation: persistentVolumeClaimTTL,
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: storageSize,
				},
			},
			VolumeMode: &volumeMode,
		},
	}

	if namespace.Storage.StorageClassName != nil && *namespace.Storage.StorageClassName != "" {
		claim.Spec.StorageClassName = namespace.Storage.StorageClassName
	}

	pvc, err := r.kubeclientset.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(context.TODO(), claim, metav1.CreateOptions{})
	if err != nil {
		log.WithFields(log.Fields{
			logfields.AerospikeCluster:      meta.Key(aerospikeCluster),
			logfields.Pod:                   meta.Key(pod),
			logfields.PersistentVolumeClaim: claim.Name,
		}).Errorf("error creating persistentvolumeclaim: %s", err)
		return nil, err
	}
	log.WithFields(log.Fields{
		logfields.AerospikeCluster:      meta.Key(aerospikeCluster),
		logfields.Pod:                   meta.Key(pod),
		logfields.PersistentVolumeClaim: claim.Name,
	}).Debug("persistentvolumeclaim created")
	return pvc, err
}

// getIndexBasedDevicePath returns the device path for the namespace
// with the specified index (e.g. 0 --> /dev/xvda, 1 --> /dev/xvdb, ...).
func getIndexBasedDevicePath(index int) string {
	return fmt.Sprintf("%s%s", defaultDevicePathPrefix, string('a'+index))
}

func (r *AerospikeClusterReconciler) signalMounted(pvc *v1.PersistentVolumeClaim) error {
	oldPVC := pvc.DeepCopy()
	removePVCAnnotation(pvc, LastUnmountedOnAnnotation)
	return r.patchPVC(oldPVC, pvc)
}

func (r *AerospikeClusterReconciler) signalUnmounted(pvc *v1.PersistentVolumeClaim) error {
	oldPVC := pvc.DeepCopy()
	setPVCAnnotation(pvc, LastUnmountedOnAnnotation, time.Now().Format(time.RFC3339))
	return r.patchPVC(oldPVC, pvc)
}

// setPVCAnnotation sets an annotation with the specified key and value in the
// aerospikeCluster object
func setPVCAnnotation(pvc *v1.PersistentVolumeClaim, key, value string) {
	if pvc.ObjectMeta.Annotations == nil {
		pvc.ObjectMeta.Annotations = make(map[string]string)
	}
	pvc.ObjectMeta.Annotations[key] = value
}

// removePVCAnnotation removes the annotation with the specified key from the
// aerospikeCluster object
func removePVCAnnotation(pvc *v1.PersistentVolumeClaim, key string) {
	delete(pvc.ObjectMeta.Annotations, key)
}

// patchCluster updates the status field of the aerospikeCluster
func (r *AerospikeClusterReconciler) patchPVC(old, new *v1.PersistentVolumeClaim) error {
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return err
	}
	newBytes, err := json.Marshal(new)
	if err != nil {
		return err
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldBytes, newBytes, &v1.PersistentVolumeClaim{})
	if err != nil {
		return err
	}
	_, err = r.kubeclientset.CoreV1().PersistentVolumeClaims(old.Namespace).Patch(context.TODO(), old.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
