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
	"os"
	"time"

	"github.com/aerospike/aerospike-client-go"
	log "github.com/sirupsen/logrus"
)

var (
	fs         *flag.FlagSet
	targetHost string
	targetPort int
)

func init() {
	fs = flag.NewFlagSet("", flag.ExitOnError)
	fs.StringVar(&targetHost, "target-host", "localhost", "the host of the target aerospike server")
	fs.IntVar(&targetPort, "target-port", 3000, "the port of the target aerospike server")
}

func main() {
	// parse the provided arguments
	fs.Parse(os.Args[1:])

	conn, err := aerospike.NewConnection(fmt.Sprintf("%s:%d", targetHost, targetPort), 100*time.Millisecond)
	if err != nil {
		log.Fatalf("failed to connect to the aerospike server: %v", err)
	}
	info, err := aerospike.RequestInfo(conn, "")
	if err != nil {
		log.Fatalf("failed to request info from the aerospike server: %v", err)
	}
	log.Info("%v", info)
}
