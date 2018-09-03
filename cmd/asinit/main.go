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
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/travelaudience/aerospike-operator/pkg/reconciler"

	log "github.com/sirupsen/logrus"
)

var (
	nodeId    string
	peerList  string
	sourceCfg string
	targetCfg string
)

func init() {
	flag.StringVar(&nodeId, "node-id", "", "the node id for the current aerospike node")
	flag.StringVar(&peerList, "peer-list", "", "comma-separated list of peers for the current aerospike node")
	flag.StringVar(&sourceCfg, "source-config", "", "path to the source configuration file")
	flag.StringVar(&targetCfg, "target-config", "", "path to the target configuration file")
}

// asinit takes a node id and a list of peers for a given aerospike
// node and updates the source configuration file with these values.
// this allows for setting node-specific configuration parameter
// which can't be set using the common configmap.
func main() {
	// parse the configuration flags
	flag.Parse()

	// read the contents of the source configuration file
	input, err := ioutil.ReadFile(sourceCfg)
	if err != nil {
		log.Fatalf("failed to read source configuration file: %v", err)
	}

	// create a split function that returns an empty slice
	// when the peerList is empty
	splitFn := func(c rune) bool {
		return c == ','
	}

	// build the list of peers
	var peers strings.Builder
	for _, peer := range strings.FieldsFunc(peerList, splitFn) {
		peers.WriteString(fmt.Sprintf("mesh-seed-address-port %s %d", peer, reconciler.HeartbeatPort))
		peers.WriteString("\n")
	}

	// replace the required placeholders in the source config
	cfg := string(input)
	cfg = strings.Replace(cfg, reconciler.ServiceNodeIdValue, nodeId, -1)
	cfg = strings.Replace(cfg, reconciler.HeartbeatAddressesValue, peers.String(), -1)

	// create the target configuration file
	if err := ioutil.WriteFile(targetCfg, []byte(cfg), 0777); err != nil {
		log.Fatalf("failed to create target configuration file: %v", err)
	}
}
