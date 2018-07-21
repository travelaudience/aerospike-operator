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

package main

import (
	"flag"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"github.com/travelaudience/aerospike-operator/pkg/admission"
	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikescheme "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned/scheme"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	"github.com/travelaudience/aerospike-operator/pkg/controller"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/debug"
	"github.com/travelaudience/aerospike-operator/pkg/signals"
	"github.com/travelaudience/aerospike-operator/pkg/versioning"
)

var (
	fs         *flag.FlagSet
	kubeconfig string
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&debug.DebugEnabled, "debug", false, "Whether to enable debug mode.")
	fs.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	fs.BoolVar(&admission.Enabled, "admission-enabled", true, "Whether to enable the validating admission webhook.")
}

func main() {
	fs.Parse(os.Args[1:])

	// workaround for https://github.com/kubernetes/kubernetes/issues/17162
	flag.CommandLine.Parse([]string{})

	// set up signals so we handle the first shutdown signal gracefully
	shCh := signals.SetupSignalHandler()

	if debug.DebugEnabled {
		log.SetLevel(log.DebugLevel)
	}
	log.WithFields(log.Fields{
		"version": versioning.OperatorVersion,
	}).Infof("aerospike-operator is starting")

	// grab the name of the current namespace so we can do leader election
	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		log.Fatalf("POD_NAMESPACE must be set")
	}
	// grab the name of the current pod so we can do leader election
	name := os.Getenv("POD_NAME")
	if name == "" {
		log.Fatalf("POD_NAME must be set")
	}
	// grab the hostname of the current pod so we can do leader election
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("failed to get hostname: %v", err)
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("error building kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("error building kubernetes clientset: %v", err)
	}

	aerospikeClient, err := aerospikeclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create aerospike clientset: %v", err)
	}

	// register (if enabled) and run the validating admission webhook and health
	// endpoint
	wh := admission.NewValidatingAdmissionWebhook(namespace, kubeClient, aerospikeClient)
	if err := wh.Register(); err != nil {
		log.Fatalf("failed to register admission webhook: %v", err)
	}
	go wh.Run(shCh)

	log.Info("attempting to become leader")

	// setup a resourcelock for leader election
	rl, _ := resourcelock.New(
		resourcelock.EndpointsResourceLock,
		namespace,
		"aerospike-operator",
		kubeClient.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      hostname,
			EventRecorder: createRecorder(kubeClient, name, namespace),
		},
	)
	// run leader election
	leaderelection.RunOrDie(leaderelection.LeaderElectionConfig{
		Lock:          rl,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leCh <-chan struct{}) {
				log.Info("started leading")
				// stop the controllers when either leCh or shCh are closed
				stopCh := make(chan struct{})
				go func() {
					select {
					case <-leCh:
						close(stopCh)
					case <-shCh:
						close(stopCh)
					}
				}()
				run(stopCh, cfg, kubeClient, aerospikeClient)
			},
			OnStoppedLeading: func() {
				log.Fatalf("stopped leading")
			},
			OnNewLeader: func(id string) {
				log.Infof("current leader: %s", id)
			},
		},
	})
}

func createRecorder(kubeClient kubernetes.Interface, name, namespace string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(log.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: typedcorev1.New(kubeClient.Core().RESTClient()).Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, v1.EventSource{Component: name})
}

func run(stopCh chan struct{}, cfg *restclient.Config, kubeClient *kubernetes.Clientset, aerospikeClient *aerospikeclientset.Clientset) {
	extsClient, err := extsclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("failed to create apiextensions clientset: %v", err)
	}
	if err := crd.NewCRDRegistry(extsClient, aerospikeClient).RegisterCRDs(); err != nil {
		log.Fatalf("failed to create custom resource definitions: %v", err)
	}

	aerospikescheme.AddToScheme(scheme.Scheme)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	aerospikeInformerFactory := aerospikeinformers.NewSharedInformerFactory(aerospikeClient, time.Second*30)

	clusterController := controller.NewAerospikeClusterController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)
	backupController := controller.NewAerospikeNamespaceBackupController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)
	restoreController := controller.NewAerospikeNamespaceRestoreController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)

	// start the shared informer factories
	go kubeInformerFactory.Start(stopCh)
	go aerospikeInformerFactory.Start(stopCh)

	// start the controllers
	var wg sync.WaitGroup
	controllers := []controller.Controller{clusterController, backupController, restoreController}
	for _, c := range controllers {
		wg.Add(1)
		go func(c controller.Controller) {
			if err := c.Run(stopCh); err != nil {
				log.Error(err)
			}
			wg.Done()
		}(c)
	}

	// wait for controllers to stop
	wg.Wait()

	// confirm successful shutdown
	log.WithFields(log.Fields{
		"version": versioning.OperatorVersion,
	}).Infof("aerospike-operator has been shut down")

	// there is a goroutine in the background that is trying to renew the leader
	// election lock. as such we must manually exit now that we know controllers
	// have been shutdown properly.
	os.Exit(0)
}
