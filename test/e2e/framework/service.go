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

package framework

import (
	"fmt"
	"net"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

const (
	servicePortName = "service"
	servicePort     = 3000
)

var (
	NodeAddress string
)

func (tf *TestFramework) CreateNodePortService(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) (*v1.Service, error) {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-nodeport", aerospikeCluster.Name),
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
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
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       servicePortName,
					Port:       servicePort,
					TargetPort: intstr.IntOrString{StrVal: servicePortName},
				},
			},
			Selector: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
			},
			Type: v1.ServiceTypeNodePort,
		},
	}
	svc, err := tf.KubeClient.CoreV1().Services(aerospikeCluster.Namespace).Create(service)
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, err
	}

	// wait for the port to be announced
	w, err := tf.KubeClient.CoreV1().Services(aerospikeCluster.Namespace).Watch(listoptions.ObjectByName(svc.Name))
	if err != nil {
		return nil, err
	}
	last, err := watch.Until(watchTimeout, w, func(event watch.Event) (bool, error) {
		obj := event.Object.(*v1.Service)
		return len(obj.Spec.Ports) >= 1, nil
	})
	if err != nil {
		return nil, err
	}
	if last == nil {
		return nil, fmt.Errorf("no events received for service %s", meta.Key(svc.Name))
	}

	return last.Object.(*v1.Service), nil
}

func (tf *TestFramework) WaitForNodePortService(svc *v1.Service) error {
	host := NodeAddress
	port := svc.Spec.Ports[0].NodePort
	addr := fmt.Sprintf("%s:%d", host, port)
	return retry(5*time.Second, 20, func() (bool, error) {
		c, err := net.DialTimeout("tcp", addr, 2500*time.Millisecond)
		if err != nil {
			return false, err
		}
		c.Close()
		return true, nil
	})
}
