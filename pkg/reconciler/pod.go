/*
Copyright 2018 The aerospike-controller Authors.

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
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/debug"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	asstrings "github.com/travelaudience/aerospike-operator/pkg/utils/strings"
)

func (r *AerospikeClusterReconciler) ensureSize(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	pods, err := r.listClusterPods(aerospikeCluster)
	if err != nil {
		return err
	}

	// grab the current and desired size of the cluster
	currentSize := len(pods)
	desiredSize := aerospikeCluster.Spec.NodeCount

	// compare the current and desired cluster size and act accordingly
	switch {
	case currentSize < desiredSize:
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.CurrentSize:      currentSize,
			logfields.DesiredSize:      desiredSize,
		}).Debug("must scale up")

		if err := r.scaleUp(aerospikeCluster, currentSize, desiredSize); err != nil {
			return err
		}

	case currentSize > desiredSize:
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.CurrentSize:      currentSize,
			logfields.DesiredSize:      desiredSize,
		}).Debug("must scale down")

		if err := r.scaleDown(aerospikeCluster, currentSize, desiredSize); err != nil {
			return err
		}

	default:
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.CurrentSize:      currentSize,
			logfields.DesiredSize:      desiredSize,
		}).Debug("no need to scale")
	}
	return nil
}

func (r *AerospikeClusterReconciler) listClusterPods(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) ([]*v1.Pod, error) {
	pods, err := r.podsLister.Pods(aerospikeCluster.Namespace).List(selectors.PodsByClusterName(aerospikeCluster.Name))
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func (r *AerospikeClusterReconciler) newPodIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) int {
	pods, err := r.listClusterPods(aerospikeCluster)
	if err != nil {
		return -1
	}
	if len(pods) == 0 {
		return 0
	}

	idxList := make([]int, 0)
	for _, p := range pods {
		idxList = append(idxList, podIndex(p.Name, aerospikeCluster.Name))
	}

	sort.Ints(idxList)
	for i := 0; i < len(idxList); i++ {
		if i != idxList[i] {
			return i
		}
	}

	return len(idxList)
}

func (r *AerospikeClusterReconciler) createPod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	configMap, err := r.getConfigMap(aerospikeCluster)
	if err != nil {
		return err
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%d", aerospikeCluster.Name, r.newPodIndex(aerospikeCluster)),
			Labels: map[string]string{
				selectors.LabelAppKey:     selectors.LabelAppVal,
				selectors.LabelClusterKey: aerospikeCluster.Name,
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
			Annotations: map[string]string{
				configMapHashLabel: asstrings.Hash(configMap.Data[configFileName]),
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "aerospike-server",
					Image: fmt.Sprintf("aerospike/aerospike-server:%s", aerospikeCluster.Spec.Version),
					Command: []string{
						"/usr/bin/asd",
						"--foreground",
						"--config-file",
						"/opt/aerospike/etc/aerospike.conf",
					},
					Ports: []v1.ContainerPort{
						{
							Name:          servicePortName,
							ContainerPort: servicePort,
						},
						{
							Name:          heartbeatPortName,
							ContainerPort: heartbeatPort,
						},
						{
							Name:          fabricPortName,
							ContainerPort: fabricPort,
						},
						{
							Name:          infoPortName,
							ContainerPort: infoPort,
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configMountPath,
						},
					},
					LivenessProbe: &v1.Probe{
						Handler: v1.Handler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.IntOrString{
									IntVal: servicePort,
								},
							},
						},
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       15,
						FailureThreshold:    3,
					},
					ReadinessProbe: &v1.Probe{
						Handler: v1.Handler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/healthz",
								Port: intstr.IntOrString{
									IntVal: asprobePort,
								},
							},
						},
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       15,
						FailureThreshold:    3,
					},
				},
				{
					Name:            "asprobe",
					Image:           "quay.io/travelaudience/aerospike-operator-tools:latest",
					ImagePullPolicy: v1.PullAlways,
					Command: []string{
						"asprobe",
						"-debug",
						"-discovery-svc", fmt.Sprintf("%s-%s", aerospikeCluster.Name, discoveryServiceSuffix),
						"-port", strconv.Itoa(asprobePort),
						"-target-port", strconv.Itoa(servicePort),
					},
					Ports: []v1.ContainerPort{
						{
							Name:          "asprobe",
							ContainerPort: asprobePort,
						},
					},
					LivenessProbe: &v1.Probe{
						Handler: v1.Handler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.IntOrString{
									IntVal: asprobePort,
								},
							},
						},
						InitialDelaySeconds: 3,
						TimeoutSeconds:      2,
						PeriodSeconds:       15,
						FailureThreshold:    3,
					},
				},
				{
					Name:            "asprom",
					Image:           "quay.io/travelaudience/aerospike-operator-tools:latest",
					ImagePullPolicy: v1.PullAlways,
					Command: []string{
						"asprom",
					},
					Ports: []v1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: aspromPort,
						},
					},
					LivenessProbe: &v1.Probe{
						Handler: v1.Handler{
							HTTPGet: &v1.HTTPGetAction{
								Path: "/metrics",
								Port: intstr.IntOrString{
									IntVal: aspromPort,
								},
							},
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: configMap.Name,
							},
						},
					},
				},
			},
		},
	}

	// Only enable in production, so it can be used in 1 node clusters while debugging (minikube)
	if !debug.DebugEnabled {
		pod.Spec.Affinity = &v1.Affinity{
			PodAntiAffinity: &v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      selectors.LabelAppKey,
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{selectors.LabelAppVal},
								},
								{
									Key:      selectors.LabelClusterKey,
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{aerospikeCluster.Name},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		}
	}

	// create the pod
	res, err := r.kubeclientset.CoreV1().Pods(aerospikeCluster.Namespace).Create(pod)
	if err != nil {
		return nil
	}

	// watch the pod, waiting for it to enter the RUNNING state
	w, err := r.kubeclientset.CoreV1().Pods(res.Namespace).Watch(listoptions.ObjectByName(res.Name))
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
		return fmt.Errorf("no events received for pod %s", meta.Key(res))
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Pod:              meta.Key(res),
	}).Debug("pod is now running")

	return nil
}

func (r *AerospikeClusterReconciler) deletePod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod) error {
	err := r.kubeclientset.CoreV1().Pods(aerospikeCluster.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: pointers.NewInt64FromFloat64(terminationGracePeriod.Seconds()),
	})
	if err != nil {
		return err
	}

	// watch the pod, waiting for it to be deleted
	w, err := r.kubeclientset.CoreV1().Pods(pod.Namespace).Watch(listoptions.ObjectByName(pod.Name))
	if err != nil {
		return err
	}
	last, err := watch.Until(watchTimeout, w, func(event watch.Event) (bool, error) {
		return event.Type == watch.Deleted, nil
	})
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for pod %s", meta.Key(pod))
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Pod:              meta.Key(pod),
	}).Debug("pod has been deleted")

	return nil
}

func (r *AerospikeClusterReconciler) scaleUp(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, currentSize, desiredSize int) error {
	for i := currentSize; i < desiredSize; i++ {
		if err := r.createPod(aerospikeCluster); err != nil {
			return err
		}
	}
	return nil
}

func (r *AerospikeClusterReconciler) scaleDown(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, currentSize, desiredSize int) error {
	pods, err := r.listClusterPods(aerospikeCluster)
	if err != nil {
		return err
	}
	sort.Sort(byIndex(pods))
	for i := currentSize; i > desiredSize; i-- {
		if err := r.deletePod(aerospikeCluster, pods[i-1]); err != nil {
			return err
		}
	}
	return nil
}

type byIndex []*v1.Pod

func (p byIndex) Len() int {
	return len(p)
}

func (p byIndex) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p byIndex) Less(i, j int) bool {
	idx1 := podIndex(p[i].Name, p[i].ObjectMeta.Labels[selectors.LabelClusterKey])
	idx2 := podIndex(p[j].Name, p[j].ObjectMeta.Labels[selectors.LabelClusterKey])
	return idx1 < idx2
}

func podIndex(podName, clusterName string) int {
	res, err := strconv.Atoi(strings.TrimPrefix(podName, fmt.Sprintf("%s-", clusterName)))
	if err != nil {
		return -1
	}
	return res
}
