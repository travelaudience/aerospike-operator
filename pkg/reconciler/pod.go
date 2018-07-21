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
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/debug"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/pointers"
	"github.com/travelaudience/aerospike-operator/pkg/utils/events"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
	"github.com/travelaudience/aerospike-operator/pkg/versioning"
)

const (
	// nodeIdPrefix is used as the prefix for node IDs so that they don't begin
	// with a leading zero. "a" stands for aerospike.
	nodeIdPrefix = "a"
)

func (r *AerospikeClusterReconciler) ensurePods(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, upgrade bool) error {
	// list existing pods for the cluster
	pods, err := r.listClusterPods(aerospikeCluster)
	if err != nil {
		return err
	}
	// grab the current and desired size of the cluster
	currentSize := len(pods)
	desiredSize := int(aerospikeCluster.Spec.NodeCount)

	// scale down if necessary
	for i := currentSize - 1; i >= desiredSize; i-- {
		if err := r.safeDeletePodWithIndex(aerospikeCluster, i); err != nil {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			}).Errorf("failed to delete pod with index %d: %v", i, err)
			return err
		}
	}

	// re-list existing pods for the cluster after scaling down
	pods, err = r.listClusterPods(aerospikeCluster)
	if err != nil {
		return err
	}
	// re-grab the current and desired size of the cluster
	currentSize = len(pods)
	desiredSize = int(aerospikeCluster.Spec.NodeCount)

	// create/upgrade/restart existing pods as required
	for i := 0; i < desiredSize; i++ {
		// attempt to grab the pod with the specified index
		pod, err := r.getPodWithIndex(aerospikeCluster, i)
		if err != nil {
			// we've failed to get the pod with the specified index
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
				logfields.PodIndex:         i,
			}).Errorf("failed to get pod: %v", err)
			// propagate the error
			return err
		}
		// check whether the pod needs to be created
		if pod == nil {
			// no pod with the specified index exists, so it must be created
			if _, err := r.createPodWithIndex(aerospikeCluster, configMap, i); err != nil {
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: meta.Key(aerospikeCluster),
					logfields.PodIndex:         i,
				}).Errorf("failed to create pod: %v", err)
				return err
			}
			// proceed to the next index
			continue
		}
		// check whether the pod needs to be upgraded
		if upgrade {
			if err := r.maybeUpgradePodWithIndex(aerospikeCluster, configMap, i); err != nil {
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: meta.Key(aerospikeCluster),
					logfields.Pod:              meta.Key(pod),
				}).Errorf("failed to upgrade pod: %v", err)
				return err
			}
			// proceed to the next index
			continue
		}
		// check whether the pod needs to be restarted
		if configMap.Annotations[configMapHashAnnotation] != pod.Annotations[configMapHashAnnotation] {
			if _, err := r.safeRestartPodWithIndex(aerospikeCluster, configMap, i); err != nil {
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: meta.Key(aerospikeCluster),
					logfields.Pod:              meta.Key(pod),
				}).Errorf("failed to restart pod: %v", err)
				return err
			}
			// proceed to the next index
			continue
		}
	}

	// signal that we're good and return
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.CurrentSize:      currentSize,
		logfields.DesiredSize:      desiredSize,
	}).Debug("pods are up-to-date")

	return nil
}

func (r *AerospikeClusterReconciler) listClusterPods(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) ([]*v1.Pod, error) {
	// read the list of pods from the lister
	pods, err := r.podsLister.Pods(aerospikeCluster.Namespace).List(selectors.PodsByClusterName(aerospikeCluster.Name))
	if err != nil {
		return nil, err
	}
	// sort the pods by index
	sort.Sort(byIndex(pods))
	// return the list of pods
	return pods, nil
}

func (r *AerospikeClusterReconciler) createPodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, index int) (*v1.Pod, error) {
	// initialConfigFilePath contains the path to the aerospike.conf file that
	// will be created as a result of mounting the configmap (i.e. before
	// templating)
	initialConfigFilePath := path.Join(initialConfigMountPath, configFileName)
	// finalConfigFilePath contains the path to the aerospike.conf file that
	// will be used by the aerospike process (i.e. after templating)
	finalConfigFilePath := path.Join(finalConfigMountPath, configFileName)
	// podName contains the name of the pod
	podName := fmt.Sprintf("%s-%d", aerospikeCluster.Name, index)
	// nodeId will contain the value used as service.node-id for the pod
	nodeId, err := computeNodeId(podName)
	if err != nil {
		return nil, fmt.Errorf("failed to compute node id for %s: %v", podName, err)
	}

	// pod represents the pod that will be created
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
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
			Annotations: map[string]string{
				configMapHashAnnotation: configMap.Annotations[configMapHashAnnotation],
				nodeIdAnnotation:        nodeId,
			},
		},
		Spec: v1.PodSpec{
			// use a init container to set the value of service.node-id to the
			// value of nodeId
			InitContainers: []v1.Container{
				{
					Name:  "init",
					Image: "busybox:1.29.1",
					Command: []string{
						"sh",
						"-c",
						fmt.Sprintf("sed 's/%s/%s/' %s>%s", serviceNodeIdValue, nodeId, initialConfigFilePath, finalConfigFilePath),
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      initialConfigVolumeName,
							MountPath: initialConfigMountPath,
						},
						{
							Name:      finalConfigVolumeName,
							MountPath: finalConfigMountPath,
						},
					},
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse(initContainerCpuRequest),
							v1.ResourceMemory: resource.MustParse(initContainerMemoryRequest),
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:  "aerospike-server",
					Image: fmt.Sprintf("aerospike/aerospike-server:%s", aerospikeCluster.Spec.Version),
					Command: []string{
						"/usr/bin/asd",
						"--foreground",
						"--config-file",
						finalConfigFilePath,
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
							Name:      finalConfigVolumeName,
							MountPath: finalConfigMountPath,
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
							v1.ResourceCPU:    computeCpuRequest(aerospikeCluster),
							v1.ResourceMemory: computeMemoryRequest(aerospikeCluster),
						},
					},
				},
				{
					Name:            "asprom",
					Image:           fmt.Sprintf("%s:%s", "quay.io/travelaudience/aerospike-operator-tools", versioning.OperatorVersion),
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
					Name: initialConfigVolumeName,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: configMap.Name,
							},
						},
					},
				},
				{
					Name: finalConfigVolumeName,
					VolumeSource: v1.VolumeSource{
						EmptyDir: &v1.EmptyDirVolumeSource{},
					},
				},
			},
			// let the reconcile loop handle pod restarts
			RestartPolicy: v1.RestartPolicyNever,
		},
	}

	// only enable in production, so it can be used in 1 node clusters while debugging (minikube)
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
		return nil, err
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
		return nil, err
	}

	done := make(chan bool, 1)
	go func() {
		ticker := time.NewTicker(podOperationFeedbackPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeStarting,
					"waiting for aerospike to start on pod %s", meta.Key(res))
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: res.Labels[selectors.LabelClusterKey],
					logfields.Pod:              meta.Key(res),
				}).Infof("waiting for aerospike to start on pod %s", meta.Key(res))
			case success := <-done:
				if success {
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonNodeStarted,
						"aerospike started on pod %s", meta.Key(res))
					log.WithFields(log.Fields{
						logfields.AerospikeCluster: res.Labels[selectors.LabelClusterKey],
						logfields.Pod:              meta.Key(res),
					}).Infof("aerospike started on pod %s", meta.Key(res))
				} else {
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeWarning, events.ReasonNodeStartedFailed,
						"could not start aerospike on pod %s", meta.Key(res))
					log.WithFields(log.Fields{
						logfields.AerospikeCluster: res.Labels[selectors.LabelClusterKey],
						logfields.Pod:              meta.Key(res),
					}).Infof("could not start aerospike on pod %s", meta.Key(res))
				}
				return
			}
		}
	}()

	currentPod := res
	err = r.waitForPodCondition(res, func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Error:
			return false, fmt.Errorf("got event of type error: %+v", event.Object)
		case watch.Deleted:
			currentPod = event.Object.(*v1.Pod)
			return false, fmt.Errorf("pod %s has been deleted", meta.Key(currentPod))
		default:
			currentPod = event.Object.(*v1.Pod)
			if isPodInTerminalState(currentPod) {
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: meta.Key(aerospikeCluster),
					logfields.Pod:              meta.Key(currentPod),
				}).Warn("pod is in terminal state")
				if err := r.deletePod(aerospikeCluster, currentPod); err != nil {
					return false, err
				}
				return false, fmt.Errorf("pod %s in terminal state has been deleted", meta.Key(currentPod))

			}
			return isPodRunningAndReady(currentPod), nil
		}
	}, watchCreatePodTimeout)
	done <- err == nil
	close(done)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
		logfields.Pod:              meta.Key(currentPod),
	}).Debug("pod created and running")

	return currentPod, nil
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

func (r *AerospikeClusterReconciler) getPodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, index int) (*v1.Pod, error) {
	// look for the pod with the specified index
	p, err := r.podsLister.Pods(aerospikeCluster.Namespace).Get(fmt.Sprintf("%s-%d", aerospikeCluster.Name, index))
	if err != nil {
		if !errors.IsNotFound(err) {
			// the pod may exist but we couldn't list it
			return nil, err
		}
		// the pod doesn't exist, so return nil but don't propagate the error
		return nil, nil
	}
	// return the pod returned by the lister
	return p, nil
}

func (r *AerospikeClusterReconciler) safeDeletePodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, index int) error {
	// check whether a pod with the specified index exists
	pod, err := r.getPodWithIndex(aerospikeCluster, index)
	if err != nil {
		// we've failed to get the pod with the specified index
		return err
	}
	if pod == nil {
		// no pod with the specified index exists
		return nil
	}
	// check whether the pod is participating in migrations
	migrations, err := podHasMigrationsInProgress(pod)
	if err != nil {
		return err
	}
	// if the pod is participating in migrations, we wait for them to finish and
	// keep on giving feedback
	if migrations {
		done := make(chan bool, 1)
		go func() {
			ticker := time.NewTicker(podOperationFeedbackPeriod)
			defer ticker.Stop()
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: pod.Labels[selectors.LabelClusterKey],
				logfields.Pod:              meta.Key(pod),
			}).Info("waiting for migrations to finish")
			r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonWaitForMigrationsStarted,
				"waiting for migrations to finish on pod %s",
				meta.Key(pod),
			)
			for {
				select {
				case <-ticker.C:
					log.WithFields(log.Fields{
						logfields.AerospikeCluster: pod.Labels[selectors.LabelClusterKey],
						logfields.Pod:              meta.Key(pod),
					}).Info("waiting for migrations to finish")
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonWaitingForMigrations,
						"waiting for migrations to finish on pod %s",
						meta.Key(pod),
					)
				case <-done:
					log.WithFields(log.Fields{
						logfields.AerospikeCluster: pod.Labels[selectors.LabelClusterKey],
						logfields.Pod:              meta.Key(pod),
					}).Info("migrations finished")
					r.recorder.Eventf(aerospikeCluster, v1.EventTypeNormal, events.ReasonWaitForMigrationsFinished,
						"migrations finished on pod %s",
						meta.Key(pod),
					)
					return
				}
			}
		}()
		if err := waitForMigrationsToFinishOnPod(pod); err != nil {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: pod.Labels[selectors.LabelClusterKey],
				logfields.Pod:              meta.Key(pod),
			}).Error("failed to wait for migrations to finish")
			return err
		}
		close(done)
	}
	// delete the pod now that migrations are finished
	return r.deletePod(aerospikeCluster, pod)
}

func (r *AerospikeClusterReconciler) safeRestartPodWithIndex(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, configMap *v1.ConfigMap, index int) (*v1.Pod, error) {
	log.WithFields(log.Fields{
		logfields.AerospikeCluster: meta.Key(aerospikeCluster),
	}).Debugf("restarting the pod with index %d", index)

	if err := r.safeDeletePodWithIndex(aerospikeCluster, index); err != nil {
		return nil, err
	}
	return r.createPodWithIndex(aerospikeCluster, configMap, index)
}

// computeCpuRequest computes the amount of cpu to be requested for the aerospike-server container and returns the
// corresponding resource.Quantity. It currently returns aerospikeServerContainerDefaultCpuRequest parsed as a quantity,
// but this may change in the future.
func computeCpuRequest(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) resource.Quantity {
	return resource.MustParse(strconv.Itoa(aerospikeServerContainerDefaultCpuRequest))
}

// computeMemoryRequest computes the amount of memory to be requested for the aerospike-server container based on the
// value of the memorySize field of each namespace, and returns the corresponding resource.Quantity.
func computeMemoryRequest(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) resource.Quantity {
	sum := 0
	for _, ns := range aerospikeCluster.Spec.Namespaces {
		if ns.MemorySize == nil {
			// ns.MemorySize is nil, which means we need to set a value that
			// matches the aerospike default for namespace.memory-size
			sum += aerospikeServerContainerDefaultMemoryRequestGi
			continue
		}
		if s, err := strconv.Atoi(strings.TrimSuffix(*ns.MemorySize, "G")); err == nil {
			// *ns.MemorySize was parsed successfully, so we use its value
			sum += s
		} else {
			log.WithFields(log.Fields{
				logfields.AerospikeCluster: meta.Key(aerospikeCluster),
			}).Warn("failed to parse memory size for namespace %s: %v", err)
			// ns.MemorySize has been validated before, so it is highly unlikely
			// than an error occurs at this point. however, if it does occur, we
			// must return something, and so we pick the default memory request.
			sum += aerospikeServerContainerDefaultMemoryRequestGi
		}
	}
	return resource.MustParse(fmt.Sprintf("%dGi", sum))
}

// computeNodeId computes the value to be used as the id of the aerospike node
// that corresponds to podName.
func computeNodeId(podName string) (string, error) {
	// calculate the md5 hash of podName
	podHash := md5.New()
	_, err := io.WriteString(podHash, podName)
	if err != nil {
		return "", err
	}
	// an aerospike node's id cannot exceed 16 characters, so we use the first
	// 15 characters of the hash and a prefix to prevent the generated id from
	// having leading zeros (which aerospike strips, causing trouble later on)
	return fmt.Sprintf("%s%s", nodeIdPrefix, hex.EncodeToString(podHash.Sum(nil))[0:15]), nil
}
