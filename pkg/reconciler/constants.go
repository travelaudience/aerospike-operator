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
	// the name of the volume that will contain the original aerospike.conf
	// created as a result of mounting the configmap (i.e. before templating)
	initialConfigVolumeName = "aerospike-conf-src"
	// the mount path of the volume that will contain the initial
	// aerospike.conf created as a result of mounting the configmap
	initialConfigMountPath = "/aerospike-conf-src"
	// the name of the volume that will contain the final aerospike.conf file
	// (i.e. after templating)
	finalConfigVolumeName = "aerospike-conf"
	// the mount path of the volume that will contain the final aerospike.conf
	// file (i.e. after templating)
	finalConfigMountPath = "/aerospike-conf"
	// the name of the aerospike.conf file
	configFileName = "aerospike.conf"

	namespaceVolumePrefix = "data-ns"

	ServicePort       = 3000
	servicePortName   = "service"
	heartbeatPort     = 3002
	heartbeatPortName = "heartbeat"
	fabricPort        = 3001
	fabricPortName    = "fabric"
	infoPort          = 3003
	infoPortName      = "info"

	watchCreatePodTimeout  = 3 * time.Hour
	watchDeletePodTimeout  = 3 * time.Minute
	terminationGracePeriod = 2 * time.Minute
	waitMigrationsTimeout  = 1 * time.Hour

	podOperationFeedbackPeriod = 2 * time.Minute
	aerospikeClientTimeout     = 10 * time.Second

	// the name of the annotation that holds the hash of the mounted configmap
	configMapHashAnnotation = "aerospike.travelaudience.com/config-map-hash"
	// the name of the annotation that holds the aerospike node id
	nodeIdAnnotation = "aerospike.travelaudience.com/node-id"
	// the name of the annotation that holds the hash of the mesh as we know it
	meshDigestAnnotation = "aerospike.travelaudience.com/mesh-hash"
	// the name of the annotation that holds the name of the pod with which a
	// PVC is associated
	PodAnnotation = "aerospike.travelaudience.com/pod-name"
	// the name of the annotation that holds the persistentVolumeClaimTTL of a
	// PVC
	PVCTTLAnnotation = "aerospike.travelaudience.com/pvc-ttl"
	// the name of the annotation that holds the timestamp at which a PVC
	// was last unmounted from a pod
	LastUnmountedOnAnnotation = "aerospike.travelaudience.com/last-unmounted-on"

	// the name of the key that corresponds to the service.node-id property
	// (used for templating)
	serviceNodeIdKey = "nodeId"
	// the value of the key that corresponds to the service.node-id property
	// (used for templating)
	serviceNodeIdValue    = "__SERVICE__NODE_ID__"
	clusterMeshServiceKey = "meshAddress"
	clusterMeshPortKey    = "meshPort"
	clusterNamespacesKey  = "namespaces"

	defaultFilePath         = "/opt/aerospike/data/"
	defaultDevicePathPrefix = "/dev/xvd"

	nsNameKey              = "name"
	nsReplicationFactorKey = "replicationFactor"
	nsMemorySizeKey        = "memorySize"
	nsDefaultTTLKey        = "defaultTTL"
	nsStorageTypeKey       = "storageType"
	nsStorageSizeKey       = "storageSize"
	nsFilePath             = "filePath"
	nsDevicePath           = "devicePath"

	aspromPortName      = "prometheus"
	aspromPort          = 9145
	aspromCpuRequest    = "10m"
	aspromMemoryRequest = "32Mi"

	asReadinessInitialDelaySeconds = 3
	asReadinessTimeoutSeconds      = 2
	asReadinessPeriodSeconds       = 10
	asReadinessFailureThreshold    = 3

	// the cpu request for the init container
	initContainerCpuRequest = "10m"
	// the memory request for the init container
	initContainerMemoryRequest = "32Mi"
	// the default cpu request for the aerospike-server container
	aerospikeServerContainerDefaultCpuRequest = 1
	// the default memory request for the aerospike-server container
	// matches the default value of namespace.memory-size
	// https://www.aerospike.com/docs/reference/configuration#memory-size
	aerospikeServerContainerDefaultMemoryRequestGi = 4

	// UpgradeStatusAnnotationKey is the name of the annotation added to
	// AerospikeCluster resources that are being upgraded.
	UpgradeStatusAnnotationKey = "aerospike.travelaudience.com/upgrade-status"
	// UpgradeStatusStartedAnnotationValue is the value of the annotation added
	// to AerospikeCluster resources that are being upgrade.
	UpgradeStatusStartedAnnotationValue = "started"
	// UpgradeStatusFailedAnnotationValue is the value of the annotation added
	// to AerospikeCluster resources that have not been successfully upgraded.
	UpgradeStatusFailedAnnotationValue = "failed"
	// UpgradeStatusBackupAnnotationValue is the value of the annotation added
	// to AerospikeCluster resources that are undergoing a pre-upgrade backup.
	UpgradeStatusBackupAnnotationValue = "backup"

	// terminal state reasons when pod status is Pending
	// container image pull failed
	ReasonImagePullBackOff = "ImagePullBackOff"
	// unable to inspect image
	ReasonImageInspectError = "ImageInspectError"
	// general image pull error
	ReasonErrImagePull = "ErrImagePull"
	// get http error when pulling image from registry
	ReasonRegistryUnavailable = "RegistryUnavailable"

	// default value for persistentVolumeClaimTTL
	defaultPersistentVolumeClaimTTL = "0d"
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
	node-id {{.nodeId}}
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

		interval 100
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
