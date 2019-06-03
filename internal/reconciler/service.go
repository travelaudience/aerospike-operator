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
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	aerospikev1alpha2 "github.com/travelaudience/aerospike-operator/internal/apis/aerospike/v1alpha2"
	"github.com/travelaudience/aerospike-operator/internal/crd"
	"github.com/travelaudience/aerospike-operator/internal/logfields"
	"github.com/travelaudience/aerospike-operator/internal/meta"
	"github.com/travelaudience/aerospike-operator/internal/pointers"
	"github.com/travelaudience/aerospike-operator/internal/utils/selectors"
)

func (r *AerospikeClusterReconciler) ensureService(aerospikeCluster *aerospikev1alpha2.AerospikeCluster) error {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: aerospikeCluster.Name,
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
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
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
			},
			Ports: []v1.ServicePort{
				{
					Name:       servicePortName,
					Port:       ServicePort,
					TargetPort: intstr.IntOrString{StrVal: servicePortName},
				},
				{
					Name:       heartbeatPortName,
					Port:       HeartbeatPort,
					TargetPort: intstr.IntOrString{StrVal: heartbeatPortName},
				},
				{
					Name:       aspromPortName,
					Port:       aspromPort,
					TargetPort: intstr.IntOrString{StrVal: aspromPortName},
				},
			},
			ClusterIP: v1.ClusterIPNone,
		},
	}

	if _, err := r.kubeclientset.CoreV1().Services(aerospikeCluster.Namespace).Create(service); err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.Service:          service.Name,
		}).Debug("service already exists")
		return nil
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Service:          service.Name,
	}).Debug("service created")
	return nil
}
