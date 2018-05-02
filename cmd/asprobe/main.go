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
	"net"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/travelaudience/aerospike-operator/pkg/asutils"
)

var (
	debug        bool
	discoverySvc string
	fs           *flag.FlagSet
	port         int
	targetHost   string
	targetPort   int
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&debug, "debug", false, "whether to enable debug logging")
	fs.StringVar(&discoverySvc, "discovery-svc", "aerospike-discovery.default", "the discovery service to query")
	fs.IntVar(&port, "port", 8080, "the port in which to listen for requests")
	fs.StringVar(&targetHost, "target-host", "localhost", "the host of the target aerospike server")
	fs.IntVar(&targetPort, "target-port", 3000, "the port of the target aerospike server")
}

func main() {
	// parse the provided arguments
	fs.Parse(os.Args[1:])

	// enable verbose logging if requested
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// expose an endpoint which will respond with 200 if the targeted aerospike node is ready and 409 otherwise.
	http.HandleFunc("/healthz", func(res http.ResponseWriter, req *http.Request) {
		if isNodeReady() {
			log.Debug("the aerospike node is ready")
			res.WriteHeader(http.StatusOK)
		} else {
			log.Debug("the aerospike node is not ready")
			res.WriteHeader(http.StatusConflict)
		}
	})

	// start serving on the specified port
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// countMembersInDiscoveryService queries the provided service and returns the number of existing srv records.
func countMembersInDiscoveryService() (int, error) {
	_, records, err := net.LookupSRV("", "", discoverySvc)
	if err != nil {
		log.Debug(err)
		return 0, err
	}
	log.Debugf("desired cluster size: %d", len(records))
	return len(records), nil
}

// isNodeReady returns whether the targeted aerospike node is ready.
func isNodeReady() bool {
	currentSize, err := asutils.GetClusterSize(targetHost, targetPort)
	if err != nil {
		return false
	}
	desiredSize, err := countMembersInDiscoveryService()
	if err != nil {
		return false
	}
	return currentSize == desiredSize
}
