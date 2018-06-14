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
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/debug"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

func (r *AerospikeClusterReconciler) ensurePods(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap) error {
	// list existing pods for the cluster
	pods, err := r.listClusterPods(aerospikeCluster)
	if err != nil {
		return err
	}

	// grab the current and desired size of the cluster
	currentSize := len(pods)
	desiredSize := aerospikeCluster.Spec.NodeCount

	// sort pods by their index
	sort.Sort(byIndex(pods))

	// scale down if necessary
	for i := currentSize - 1; i >= desiredSize; i-- {
		log.WithFields(log.Fields{
			logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			logfields.CurrentSize:      currentSize,
			logfields.DesiredSize:      desiredSize,
		}).Debugf("deleting %s", meta.Key(pods[i]))

		// this pod is not needed anymore, so it must be deleted
		if err := r.safeDeletePod(aerospikeCluster, pods[i]); err != nil {
			return err
		}
	}

	for i := 0; i < desiredSize; i++ {
		if i < currentSize {
			if configMap.Annotations[configMapHashLabel] != pods[i].Annotations[configMapHashLabel] {
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: meta.Key(aerospikeCluster),
					logfields.CurrentSize:      currentSize,
					logfields.DesiredSize:      desiredSize,
				}).Debugf("restarting %s", meta.Key(pods[i]))

				// this pod must be restarted in order for aerospike to notice the new config
				if err := r.safeRestartPod(aerospikeCluster, configMap, pods[i]); err != nil {
					return err
				}
			}
		} else {
			// create a new pod in order to meet the desired size
			if err := r.createPod(aerospikeCluster, configMap); err != nil {
				return err
			}
		}
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.CurrentSize:      currentSize,
		logfields.DesiredSize:      desiredSize,
	}).Debug("pods are up-to-date")

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
		idxList = append(idxList, podIndex(p))
	}

	sort.Ints(idxList)
	for i := 0; i < len(idxList); i++ {
		if i != idxList[i] {
			return i
		}
	}

	return len(idxList)
}

func (r *AerospikeClusterReconciler) createPod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap) error {
	return r.createPodWithIndex(aerospikeCluster, configMap, r.newPodIndex(aerospikeCluster))
}

func (r *AerospikeClusterReconciler) createPodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, index int) error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%d", aerospikeCluster.Name, index),
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
				configMapHashLabel: configMap.Annotations[configMapHashLabel],
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
							ContainerPort: ServicePort,
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
					ReadinessProbe: &v1.Probe{
						Handler: v1.Handler{
							TCPSocket: &v1.TCPSocketAction{
								Port: intstr.IntOrString{
									IntVal: ServicePort,
								},
							},
						},
						InitialDelaySeconds: asReadinessInitialDelaySeconds,
						TimeoutSeconds:      asReadinessTimeoutSeconds,
						PeriodSeconds:       asReadinessPeriodSeconds,
						FailureThreshold:    asReadinessFailureThreshold,
					},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(asCpuRequest),
							v1.ResourceMemory: computeMemoryRequest(aerospikeCluster),
						},
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
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(aspromCpuRequest),
							v1.ResourceMemory: resource.MustParse(aspromMemoryRequest),
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

	for _, namespace := range aerospikeCluster.Spec.Namespaces {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      fmt.Sprintf("%s-%s", namespaceVolumePrefix, namespace.Name),
			MountPath: fmt.Sprintf("%s%s", defaultFilePath, namespace.Name),
		})
	}

	if claims, err := r.getPersistentVolumeClaims(aerospikeCluster, pod); err != nil {
		return err
	} else {
		for _, claim := range claims {
			pod.Spec.Volumes = append(pod.Spec.Volumes, v1.Volume{
				Name: fmt.Sprintf("%s-%s", namespaceVolumePrefix, claim.Labels[selectors.LabelNamespaceKey]),
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: claim.Name,
					},
				},
			})
		}
	}

	// create the pod
	res, err := r.kubeclientset.CoreV1().Pods(aerospikeCluster.Namespace).Create(pod)
	if err != nil {
		return err
	}

	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(podOperationFeedbackPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeStarting,
					"waiting for aerospike to start on pod %s",
					meta.Key(res),
				)
			case <-done:
				r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeStarted,
					"aerospike started on pod %s",
					meta.Key(res),
				)
				return
			}
		}
	}()

	err = r.waitForPodCondition(res, func(event watch.Event) (bool, error) {
		return isPodRunningAndReady(event.Object.(*v1.Pod)), nil
	}, watchCreatePodTimeout)
	if err != nil {
		return err
	}
	close(done)

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Pod:              meta.Key(res),
	}).Debug("pod created and running")

	return nil
}

func (r *AerospikeClusterReconciler) deletePod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod) error {
	err := r.kubeclientset.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{
		GracePeriodSeconds: pointers.NewInt64FromFloat64(terminationGracePeriod.Seconds()),
	})
	if err != nil {
		return err
	}

	err = r.waitForPodCondition(pod, func(event watch.Event) (bool, error) {
		return event.Type == watch.Deleted, nil
	}, watchDeletePodTimeout)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Pod:              meta.Key(pod),
	}).Debug("pod has been deleted")

	return nil
}

func (r *AerospikeClusterReconciler) safeDeletePod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, pod *v1.Pod) error {
	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(podOperationFeedbackPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonMigrationsFinishing,
					"waiting for migrations to finish on pod %s",
					meta.Key(pod),
				)
			case <-done:
				r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonMigrationsFinished,
					"migrations finished on pod %s",
					meta.Key(pod),
				)
				return
			}
		}
	}()

	if err := waitPodReadyToShutdown(pod); err != nil {
		return err
	}
	close(done)

	return r.deletePod(aerospikeCluster, pod)
}

func (r *AerospikeClusterReconciler) safeRestartPod(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, pod *v1.Pod) error {
	if err := r.safeDeletePod(aerospikeCluster, pod); err != nil {
		return err
	}
	return r.createPodWithIndex(aerospikeCluster, configMap, podIndex(pod))
}

// computeMemoryRequest computes the amount of memory to be requested per pod based on the value of the memorySize field
// of each namespace and returns the corresponding resource.Quantity.
func computeMemoryRequest(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) resource.Quantity {
	sum := 0
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		s, err := strconv.Atoi(strings.TrimSuffix(ns.MemorySize, "G"))
		if err != nil {
			// ns.MemorySize has been validated before, so it is highly unlikely
			// than an error occurs at this point. however, if it does occur, we
			// must return something and so we pick 1Gi as the default quantity.
			return resource.MustParse("1Gi")
		}
		sum += s
	}
	return resource.MustParse(fmt.Sprintf("%dGi", sum))
}
