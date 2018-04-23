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
	"text/template"
	"time"
)

const (
	kind = "AerospikeCluster"

	configVolumeName = "config"
	configMountPath  = "/opt/aerospike/etc/"

	namespaceVolumePrefix = "data-ns"

	servicePort       = 3000
	servicePortName   = "service"
	heartbeatPort     = 3002
	heartbeatPortName = "heartbeat"
	fabricPort        = 3001
	fabricPortName    = "fabric"
	infoPort          = 3003
	infoPortName      = "info"

	watchTimeout           = 3 * time.Minute
	terminationGracePeriod = 5 * time.Second

	discoveryServiceSuffix = "discovery"
	configMapSuffix        = "config"

	configMapHashLabel = "configMapHash"

	configFileName = "aerospike.conf"

	clusterMeshServiceKey = "meshAddress"
	clusterMeshPortKey    = "meshPort"
	clusterNamespacesKey  = "namespaces"

	defaultFilePath   = "/opt/aerospike/data/"
	defaultDevicePath = "/dev/xvda"

	nsNameKey              = "name"
	nsReplicationFactorKey = "replicationFactor"
	nsMemorySizeKey        = "memorySize"
	nsDefaultTTLKey        = "defaultTTL"
	nsStorageTypeKey       = "storageType"
	nsStorageSizeKey       = "storageSize"
	nsFilePath             = "filePath"
	nsDevicePath           = "devicePath"

	asprobePort    = 8080
	aspromPortName = "prometheus"
	aspromPort     = 9145
)

var asConfigTemplate = template.Must(template.New("aerospike-config").Parse(aerospikeConfig))
var asNamespaceTemplate = template.Must(template.New("as-namespace-config").Parse(aerospikeNamespaceConfig))

const aerospikeConfig = `
service {
	user root
	group root
	paxos-single-replica-limit 1
	pidfile /var/run/aerospike/asd.pid
	service-threads 4
	transaction-queues 4
	transaction-threads-per-queue 4
	proto-fd-max 15000
}

logging {
	file /var/log/aerospike/aerospike.log {
		context any info
	}

	console {
		context any info 
	}
}

network {
	service {
		address any
		port 3000
	}

	heartbeat {
		mode mesh
		port 3002

		mesh-seed-address-port {{.meshAddress}} {{.meshPort}}

		interval 150
		timeout 10
	}

	fabric {
		port 3001
	}

	info {
		port 3003
	}
}

{{range .namespaces}}
	{{.}}
{{end}}
`

const aerospikeNamespaceConfig = `
namespace {{.name}} {

	{{if .replicationFactor}}
		replication-factor {{.replicationFactor}}
	{{end}}

	{{if .memorySize}}
		memory-size {{.memorySize}}
	{{end}}

	{{if .defaultTTL}}
		default-ttl {{.defaultTTL}}
	{{end}}

	storage-engine device {

		{{if eq .storageType "file"}}
			file {{.filePath}}{{.name}}/{{.name}}.dat
		{{else if eq .storageType "device"}}
			device {{.devicePath}}
		{{end}}

		{{if .storageSize}}
			filesize {{.storageSize}}
		{{end}}
	}
}`
