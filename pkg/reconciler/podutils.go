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
	"fmt"
	"strconv"
	"strings"
	"time"

	as "github.com/aerospike/aerospike-client-go"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	watchapi "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"

	"github.com/travelaudience/aerospike-operator/pkg/meta"
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

// isPodInFailureState attempts to checks if the specified pod has reached an error condition from which it is not
// expected to recover.
func isPodInFailureState(pod *v1.Pod) bool {
	// if the value of ".status.phase" is "Failed", trhe pod is trivially in a failure state
	if pod.Status.Phase == v1.PodFailed {
		return true
	}

	// grab the status of every container in the pod (including its init containers)
	containerStatus := append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...)

	// inspect the status of each individual container for common failure states
	for _, container := range containerStatus {
		// if the container is marked as "Terminated", check if its exit code is non-zero since this may still represent
		// a container that has terminated successfully (such as an init container)
		if terminated := container.State.Terminated; terminated != nil && terminated.ExitCode != 0 {
			return true
		}
		// if the container is marked as "Waiting", check for common image-related errors
		if waiting := container.State.Waiting; waiting != nil && isImageError(waiting.Reason) {
			return true
		}
	}

	// no failure state was found
	return false
}

// isImageError indicated whether the specified reason corresponds to an error while pulling or inspecting a container
// image.
func isImageError(reason string) bool {
	return reason == ReasonErrImagePull || reason == ReasonImageInspectError || reason == ReasonImagePullBackOff || reason == ReasonRegistryUnavailable
}

func (r *AerospikeClusterReconciler) waitForPodCondition(pod *v1.Pod, fn watch.ConditionFunc, timeout time.Duration) error {
	fs := selectors.ObjectByCoordinates(pod.Namespace, pod.Name)
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fs.String()
			return r.kubeclientset.CoreV1().Pods(pod.Namespace).List(options)
		},
		WatchFunc: func(options metav1.ListOptions) (watchapi.Interface, error) {
			options.FieldSelector = fs.String()
			return r.kubeclientset.CoreV1().Pods(pod.Namespace).Watch(options)
		},
	}
	ctx, cfn := context.WithTimeout(context.Background(), timeout)
	defer cfn()
	last, err := watch.UntilWithSync(ctx, lw, &v1.Pod{}, nil, fn)
	if err != nil {
		return err
	}
	if last == nil {
		return fmt.Errorf("no events received for pod %s", meta.Key(pod))
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

func tipClearHostname(pod *v1.Pod, address string) error {
	_, err := runInfoCommandOnPod(pod, fmt.Sprintf("tip-clear:host-port-list=%s:%d", address, HeartbeatPort))
	return err
}

func alumniReset(pod *v1.Pod) error {
	_, err := runInfoCommandOnPod(pod, "services-alumni-reset")
	return err
}
