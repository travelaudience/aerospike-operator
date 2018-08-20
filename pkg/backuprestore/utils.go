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

package backuprestore

import "fmt"

const (
	// metaObjectFormatString represents the string format used by the backup tool
	// to generate the metadata file name.
	metaObjectFormatString = "%s.json"
	// backupObjectFormatString represents the string format used by the backup tool
	// to generate the backup data file name.
	backupObjectFormatString = "%s.asb.gz"
)

// GetObjectName returns the object name formatted according to
// the specified format
func GetMetadataObjectName(asNamespaceBackupName string) string {
	return fmt.Sprintf(metaObjectFormatString, asNamespaceBackupName)
}

func GetBackupObjectName(asNamespaceBackupName string) string {
	return fmt.Sprintf(backupObjectFormatString, asNamespaceBackupName)
}
