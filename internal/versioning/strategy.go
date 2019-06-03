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

package versioning

// UpgradeStrategy describes how to upgrade a pod.
type UpgradeStrategy struct {
	// RecreatePersistentVolumeClaims indicates whether new persistent
	// volume claims should be created for pods.
	RecreatePersistentVolumeClaims bool
}

var (
	// DefaultStrategy represents the strategy used for performing
	// version upgrades between versions that do not require any special
	// treatment
	DefaultStrategy = &UpgradeStrategy{
		RecreatePersistentVolumeClaims: false,
	}

	// To42XYStrategy represents the strategy used for performing
	// version upgrades from versions prior to 4.2.X.Y to 4.2.X.Y
	// (or newer)
	To42XYStrategy = &UpgradeStrategy{
		RecreatePersistentVolumeClaims: true,
	}
)
