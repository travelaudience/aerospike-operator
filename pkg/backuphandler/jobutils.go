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

import (
	"fmt"

	aerospikev1alpha1 "github.com/travelaudience/aerospike-operator/pkg/apis/aerospike/v1alpha1"
)

func pipeDirection(action aerospikev1alpha1.ActionType) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return "<"
	}
	return ">"
}

func inputOutputString(action aerospikev1alpha1.ActionType) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return "--input-file"
	}
	return "--output-file"
}

func backupCommand(action aerospikev1alpha1.ActionType) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return "restore"
	}
	return "save"
}

func metaCommand(action aerospikev1alpha1.ActionType, namespace string) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return fmt.Sprintf("OLDNAMESPACE=$(cat %s/%s)", sharedVolumeMountPath, sharedMetadataPipeName)
	}
	return fmt.Sprintf("echo %q > %s/%s", namespace, sharedVolumeMountPath, sharedMetadataPipeName)
}

func getNamespace(action aerospikev1alpha1.ActionType, namespace string) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return fmt.Sprintf("$OLDNAMESPACE,%s", namespace)
	}
	return namespace
}

func asCommand(action aerospikev1alpha1.ActionType) string {
	if action == aerospikev1alpha1.ActionTypeRestore {
		return "asrestore"
	}
	return "asbackup"
}
