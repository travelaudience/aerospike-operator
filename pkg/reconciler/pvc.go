package reconciler

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

var (
	FileSystemVolumeMode = v1.PersistentVolumeFilesystem
)

func (r *AerospikeClusterReconciler) getPersistentVolumeClaims(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod) ([]*v1.PersistentVolumeClaim, error) {
	claims := make([]*v1.PersistentVolumeClaim, len(aerospikeCluster.Spec.Namespaces))

	for i, namespace := range aerospikeCluster.Spec.Namespaces {
		storageSize, err := resource.ParseQuantity(namespace.Storage.Size)
		if err != nil {
			return nil, err
		}

		claim := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s", namespace.Name, pod.Name),
				Labels: map[string]string{
					selectors.LabelAppKey:       selectors.LabelAppVal,
					selectors.LabelNamespaceKey: namespace.Name,
					selectors.LabelClusterKey:   aerospikeCluster.Name,
				},
				Namespace: aerospikeCluster.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         aerospikev1alpha1.SchemeGroupVersion.String(),
						Kind:               kind,
						Name:               aerospikeCluster.Name,
						UID:                aerospikeCluster.UID,
						Controller:         pointers.NewBool(true),
						BlockOwnerDeletion: pointers.NewBool(true),
					},
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
				VolumeMode: &FileSystemVolumeMode,
			},
		}

		if namespace.Storage.StorageClassName != "" {
			claim.Spec.StorageClassName = &namespace.Storage.StorageClassName
		}

		pvc, err := r.kubeclientset.CoreV1().PersistentVolumeClaims(claim.Namespace).Create(claim)
		if err != nil {
			if !errors.IsAlreadyExists(err) {
				return nil, err
			}
			if pvc, err = r.pvcsLister.PersistentVolumeClaims(aerospikeCluster.Namespace).Get(claim.Name); err != nil {
				return nil, err
			}
			log.WithFields(log.Fields{
				logfields.AerospikeCluster:      meta.Key(aerospikeCluster),
				logfields.PersistentVolumeClaim: claim.Name,
			}).Debug("persistentvolumeclaim already exists")
		} else {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster:      meta.Key(aerospikeCluster),
				logfields.PersistentVolumeClaim: claim.Name,
			}).Debug("persistentvolumeclaim created")
		}
		claims[i] = pvc
	}
	return claims, nil
}
