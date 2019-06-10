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

package logfields

const (
	Kind                      = "kind"
	CurrentSize               = "currentSize"
	DesiredSize               = "desiredSize"
	AerospikeCluster          = "aerospikecluster"
	AerospikeNamespaceBackup  = "aerospikenamespacebackup"
	AerospikeNamespaceRestore = "aerospikenamespacerestore"
	Pod                       = "pod"
	Node                      = "node"
	Service                   = "service"
	ConfigMap                 = "configmap"
	PersistentVolumeClaim     = "persistentvolumeclaim"
	Key                       = "key"
	Job                       = "job"
	PodIndex                  = "podIndex"
)
