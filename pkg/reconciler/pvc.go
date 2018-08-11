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
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

var (
	volumeModeMap = map[string]v1.PersistentVolumeMode{
		aerospikev1alpha1.StorageTypeDevice: v1.PersistentVolumeBlock,
		aerospikev1alpha1.StorageTypeFile:   v1.PersistentVolumeFilesystem,
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

func (r *AerospikeClusterReconciler) getPersistentVolumeClaim(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod) (*v1.PersistentVolumeClaim, error) {
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
		if podName, ok := pvc.Annotations[podAnnotation]; ok && podName == pod.Name {
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

func (r *AerospikeClusterReconciler) createPersistentVolumeClaim(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod, namespace *aerospikev1alpha1.AerospikeNamespaceSpec) (*v1.PersistentVolumeClaim, error) {
	storageSize, err := resource.ParseQuantity(namespace.Storage.Size)
	if err != nil {
		return nil, err
	}

	volumeMode := volumeModeMap[namespace.Storage.Type]
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
					APIVersion:         aerospikev1alpha1.SchemeGroupVersion.String(),
					Kind:               crd.AerospikeClusterKind,
					Name:               aerospikeCluster.Name,
					UID:                aerospikeCluster.UID,
					Controller:         pointers.NewBool(true),
					BlockOwnerDeletion: pointers.NewBool(true),
				},
			},
			Annotations: map[string]string{
				podAnnotation: pod.Name,
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

	if namespace.Storage.StorageClassName != "" {
		claim.Spec.StorageClassName = &namespace.Storage.StorageClassName
	}

	pvc, err := r.kubeclientset.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(claim)
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
