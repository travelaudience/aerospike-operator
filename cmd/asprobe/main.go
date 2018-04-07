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

package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
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

// countMembersInAerospikeCluster queries the provided host and port and returns the cluster size reported by aerospike.
func countMembersInAerospikeCluster() (int, error) {
	conn, err := aerospike.NewConnection(fmt.Sprintf("%s:%d", targetHost, targetPort), 100*time.Millisecond)
	if err != nil {
		log.Debugf("failed to connect to the aerospike server: %v", err)
		return 0, err
	}
	info, err := aerospike.RequestInfo(conn, "")
	if err != nil {
		log.Debugf("failed to request info from the aerospike server: %v", err)
		return 0, err
	}

	str, ok := info["statistics"]
	if !ok {
		log.Debug("malformed input received from aerospike")
		return 0, err
	}

	stats := parseStatistics(str)

	str, ok = stats["cluster_size"]
	if !ok {
		log.Debug("cluster_size is not present")
		return 0, fmt.Errorf("cluster_size is not present")
	}
	size, err := strconv.Atoi(str)
	if err != nil {
		log.Debug("failed to parse cluster_size as an integer")
		return 0, err
	}

	log.Debugf("current cluster size: %d", size)
	return size, nil
}

// parseStatistics parses a string in the form a=b;c=d; into a map[string]string, trimming whitespace in the process.
func parseStatistics(stats string) map[string]string {
	res := make(map[string]string)
	pairs := strings.Split(stats, ";")
	for _, pair := range pairs {
		r := strings.Split(pair, "=")
		if len(r) == 2 {
			res[strings.TrimSpace(r[0])] = strings.TrimSpace(r[1])
		}
	}
	return res
}

// isNodeReady returns whether the targeted aerospike node is ready.
func isNodeReady() bool {
	currentSize, err := countMembersInAerospikeCluster()
	if err != nil {
		return false
	}
	desiredSize, err := countMembersInDiscoveryService()
	if err != nil {
		return false
	}
	return currentSize == desiredSize
}
