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
	"strconv"
	"strings"
	"sync"
	"time"

	as "github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
	"github.com/travelaudience/aerospike-operator/pkg/logfields"
	"github.com/travelaudience/aerospike-operator/pkg/meta"
	"github.com/travelaudience/aerospike-operator/pkg/utils/listoptions"
	"github.com/travelaudience/aerospike-operator/pkg/utils/selectors"
)

type byIndex []*v1.Pod

func (p byIndex) Len() int {
	return len(p)
}

func (p byIndex) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p byIndex) Less(i, j int) bool {
	idx1 := podIndex(p[i])
	idx2 := podIndex(p[j])
	return idx1 < idx2
}

func podIndex(pod *v1.Pod) int {
	res, err := strconv.Atoi(strings.TrimPrefix(pod.Name, fmt.Sprintf("%s-", pod.ObjectMeta.Labels[selectors.LabelClusterKey])))
	if err != nil {
		return -1
	}
	return res
}

func isPodRunningAndReady(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning && podutil.IsPodReady(pod)
}

func isPodInTerminalState(pod *v1.Pod) bool {
	// pod is in terminal state if its .status.phase is Failed, or
	// if its .status.phase is Pending and the reason is "ImagePullBackOff"
	switch pod.Status.Phase {
	case v1.PodFailed:
		return true
	case v1.PodPending:
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if waiting := containerStatus.State.Waiting; waiting != nil {
				switch waiting.Reason {
				case ReasonImagePullBackOff:
					fallthrough
				case ReasonImageInspectError:
					fallthrough
				case ReasonErrImagePull:
					fallthrough
				case ReasonRegistryUnavailable:
					return true
				}
			}
		}
	}
	return false
}

func (r *AerospikeClusterReconciler) waitForPodCondition(pod *v1.Pod, fn watch.ConditionFunc, timeout time.Duration) error {
	start := time.Now()
	w, err := r.kubeclientset.CoreV1().Pods(pod.Namespace).Watch(listoptions.ObjectByNameAndVersion(pod.Name, pod.ResourceVersion))
	if err != nil {
		return err
	}

	lastPod := pod
	last, err := watch.Until(timeout, w, fn)
	if err != nil {
		if err == watch.ErrWatchClosed {
			if t := timeout - time.Since(start); t > 0 {
				if last != nil {
					lastPod = last.Object.(*v1.Pod)
				}
				return r.waitForPodCondition(lastPod, fn, t)
			}
		}
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for %s", meta.Key(pod))
	}
	return nil
}

func podHasMigrationsInProgress(pod *v1.Pod) (bool, error) {
	client, err := as.NewClient(pod.Status.PodIP, ServicePort)
	if err != nil {
		return false, err
	}
	defer client.Close()
	// try to find the current node by its id/name
	for _, node := range client.Cluster().GetNodes() {
		// node.GetName returns an upper-case string, so we must ignore case
		if strings.EqualFold(node.GetName(), pod.Annotations[nodeIdAnnotation]) {
			return node.MigrationInProgress()
		}
	}
	return false, fmt.Errorf("failed to find node %s in the cluster", pod.Annotations[nodeIdAnnotation])
}

func waitForMigrationsToFinishOnPod(pod *v1.Pod) error {
	client, err := as.NewClient(pod.Status.PodIP, ServicePort)
	if err != nil {
		return err
	}
	defer client.Close()
	// try to find the current node by its id/name
	for _, node := range client.Cluster().GetNodes() {
		// node.GetName returns an upper-case string, so we must ignore case
		if strings.EqualFold(node.GetName(), pod.Annotations[nodeIdAnnotation]) {
			return node.WaitUntillMigrationIsFinished(waitMigrationsTimeout)
		}
	}
	return fmt.Errorf("failed to find node %s in the cluster", pod.Annotations[nodeIdAnnotation])
}

func runInfoCommandOnPod(pod *v1.Pod, command string) (map[string]string, error) {
	addr := fmt.Sprintf("%s:%d", pod.Status.PodIP, ServicePort)
	conn, err := as.NewConnection(addr, aerospikeClientTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return as.RequestInfo(conn, command)
}

func getAerospikeServerVersionFromPod(pod *v1.Pod) (string, error) {
	res, err := runInfoCommandOnPod(pod, "build")
	if err != nil {
		return "", err
	}
	version, ok := res["build"]
	if !ok {
		return "", fmt.Errorf("failed to get aerospike version from pod %v", meta.Key(pod))
	}

	return version, nil
}

func executeInfoCommandOnAllNodes(aerospikeCluster *aerospikev1alpha1.AerospikeCluster, command string) error {
	client, err := as.NewClient(fmt.Sprintf("%s.%s", aerospikeCluster.Name, aerospikeCluster.Namespace), ServicePort)
	if err != nil {
		return err
	}
	defer client.Close()

	nodes := client.GetNodes()
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	errorOccured := make(chan error, len(nodes))
	defer close(errorOccured)
	// execute the command in a goroutine
	for _, node := range nodes {
		go func(node *as.Node) {
			_, err := node.RequestInfo(command)
			if err != nil {
				errorOccured <- err
				log.WithFields(log.Fields{
					logfields.AerospikeCluster: aerospikeCluster.Name,
					logfields.Node:             node.GetName(),
				}).Infof("error executing %q on node %s: %s", command, node.GetName(), err)
			}
			wg.Done()
		}(node)
	}
	wg.Wait()
	select {
	case <-errorOccured:
		return fmt.Errorf("failed to execute command %q in one or more nodes", command)
	default:
		return nil
	}
}

func tipClearNode(pod *v1.Pod) error {
	_, err := runInfoCommandOnPod(pod, "tip-clear:host-port-list=all")
	return err
}

func addTipToNode(pod *v1.Pod) error {
	command := fmt.Sprintf("tip:host=%s;port=%d", pod.Labels[selectors.LabelClusterKey], heartbeatPort)
	_, err := runInfoCommandOnPod(pod, command)
	return err
}

func alumniResetAllNodes(aerospikeCluster *aerospikev1alpha1.AerospikeCluster) error {
	return executeInfoCommandOnAllNodes(aerospikeCluster, "services-alumni-reset")
}
