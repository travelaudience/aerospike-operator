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

package backuphandler

import "time"

const (
	bucketSecretVolumeName      = "bucket-secret-volume"
	bucketSecretVolumeMountPath = "/creds"

	operationStateKey      = "state"
	operationStateFinished = "finished"

	backupExtension = "asb.gz"
	secretFileName  = "key.json"
)

var (
	watchJobTimeout = time.Hour * 1
)

type actionType string

var (
	backupAction  actionType = "backup"
	restoreAction actionType = "restore"
)
