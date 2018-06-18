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
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

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
	fs.StringVar(&admission.ServiceName, "admission-service-name", "aerospike-operator", "The name of the service used to expose the admission webhook.")
}

func main() {
	fs.Parse(os.Args[1:])

	if debug.DebugEnabled {
		log.SetLevel(log.DebugLevel)
	}

	log.WithFields(log.Fields{
		"version": versioning.OperatorVersion,
	}).Infof("aerospike-operator is starting")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

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
		log.Fatalf("error building aerospike clientset: %v", err)
	}

	extsClient, err := extsclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("error building apiextensions clientset: %v", err)
	}

	if err := crd.NewCRDRegistry(extsClient).RegisterCRDs(); err != nil {
		log.Fatalf("error creating custom resource definitions: %v", err)
	}

	aerospikescheme.AddToScheme(scheme.Scheme)

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	aerospikeInformerFactory := aerospikeinformers.NewSharedInformerFactory(aerospikeClient, time.Second*30)

	clusterController := controller.NewAerospikeClusterController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)
	backupController := controller.NewAerospikeNamespaceBackupController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)
	restoreController := controller.NewAerospikeNamespaceRestoreController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)

	// if --admission-enabled is true create, register and run the validating admission webhook
	readyCh := make(chan interface{}, 0)
	go admission.NewValidatingAdmissionWebhook(kubeClient).RegisterAndRun(readyCh)

	// wait for the webhook to be ready to start the controllers
	<-readyCh

	// start the controllers
	var wg sync.WaitGroup
	controllers := []controller.Controller{clusterController, backupController, restoreController}
	for _, c := range controllers {
		wg.Add(1)
		go func(c controller.Controller) {
			if err := c.Run(2, stopCh); err != nil {
				log.Error(err)
			}
			wg.Done()
		}(c)
	}

	// start the shared informer factories
	go kubeInformerFactory.Start(stopCh)
	go aerospikeInformerFactory.Start(stopCh)

	log.Debug("waiting for all controllers to shut down gracefully")
	wg.Wait()
}
