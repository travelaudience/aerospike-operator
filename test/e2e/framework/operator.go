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
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

// OperatorImage is the image used to deploy aerospike-operator.
var OperatorImage string

const (
	watchTimeout = 5 * time.Minute
	// OperatorNamespace is the namespace in which to create the aerospike-operator pod.
	OperatorNamespace = "aerospike-operator"
	// OperatorNamespace is the name used for aerospike-operator pod.
	OperatorName = "aerospike-operator"

	operatorAppValue       = "aerospike-operator"
	operatorServiceAccount = "aerospike-operator"
)

func (tf *TestFramework) createOperator() error {
	if OperatorImage == "" {
		log.Warnf("no aerospike-operator image specified, assuming a local instance")
		return nil
	}

	res, err := tf.KubeClient.CoreV1().Pods(OperatorNamespace).Create(createPodObj())
	if err != nil {
		return err
	}

	w, err := tf.KubeClient.CoreV1().Pods(OperatorNamespace).Watch(listoptions.ObjectByName(OperatorName))
	if err != nil {
		return err
	}
	last, err := watch.Until(watchTimeout, w, func(event watch.Event) (bool, error) {
		return event.Object.(*v1.Pod).Status.Phase == v1.PodRunning, nil
	})
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for %s", meta.Key(res))
	}
	return nil
}

func (tf *TestFramework) deleteOperator() error {
	if OperatorImage == "" {
		return nil
	}
	return tf.KubeClient.CoreV1().Pods(OperatorNamespace).Delete(OperatorName, &metav1.DeleteOptions{})
}

func (tf *TestFramework) createOperatorService() error {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorName,
			Labels: map[string]string{
				selectors.LabelAppKey: operatorAppValue,
			},
			Namespace: OperatorNamespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				selectors.LabelAppKey: operatorAppValue,
			},
			Ports: []v1.ServicePort{
				{
					Port:       443,
					TargetPort: intstr.IntOrString{IntVal: 8443},
				},
			},
		},
	}

	if _, err := tf.KubeClient.CoreV1().Services(OperatorNamespace).Create(service); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (tf *TestFramework) deleteOperatorService() error {
	if OperatorImage == "" {
		return nil
	}
	return tf.KubeClient.CoreV1().Services(OperatorNamespace).Delete(OperatorName, &metav1.DeleteOptions{})
}

func createPodObj() *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				selectors.LabelAppKey: operatorAppValue,
			},
			Name:      OperatorName,
			Namespace: OperatorNamespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            OperatorName,
					Image:           OperatorImage,
					ImagePullPolicy: v1.PullAlways,
					Ports: []v1.ContainerPort{
						{
							ContainerPort: 8443,
						},
					},
					Env: []v1.EnvVar{
						{
							Name: "POD_NAMESPACE",
							ValueFrom: &v1.EnvVarSource{
								FieldRef: &v1.ObjectFieldSelector{
									FieldPath: "metadata.namespace",
								},
							},
						},
					},
					Command: []string{
						"aerospike-operator",
						"-debug",
					},
				},
			},
			RestartPolicy:      v1.RestartPolicyNever,
			ServiceAccountName: operatorServiceAccount,
		},
	}
}
