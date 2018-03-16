/*
Copyright 2017 The Kubernetes Authors.

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
	"time"

	log "github.com/sirupsen/logrus"
	extsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	//_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	aerospikeclientset "github.com/travelaudience/aerospike-operator/pkg/client/clientset/versioned"
	aerospikeinformers "github.com/travelaudience/aerospike-operator/pkg/client/informers/externalversions"
	"github.com/travelaudience/aerospike-operator/pkg/controller"
	"github.com/travelaudience/aerospike-operator/pkg/crd"
	"github.com/travelaudience/aerospike-operator/pkg/signals"
)

var (
	fs         *flag.FlagSet
	masterURL  string
	kubeconfig string
	debug      bool
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&debug, "debug", false, "Whether to enable debug logging.")
	fs.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {
	fs.Parse(os.Args[1:])

	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
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

	if err := crd.Ensure(extsClient); err != nil {
		log.Fatalf("error creating custom resource definition: %v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	aerospikeInformerFactory := aerospikeinformers.NewSharedInformerFactory(aerospikeClient, time.Second*30)

	clusterController := controller.NewAerospikeClusterController(kubeClient, aerospikeClient, kubeInformerFactory, aerospikeInformerFactory)

	go kubeInformerFactory.Start(stopCh)
	go aerospikeInformerFactory.Start(stopCh)

	if err = clusterController.Run(2, stopCh); err != nil {
		log.Fatalf("error running controller: %v", err)
	}
}
