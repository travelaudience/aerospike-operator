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

const (
	aerospikeServer_4_0_0_4  = "4.0.0.4"
	aerospikeServer_4_0_0_5  = "4.0.0.5"
	aerospikeServer_4_0_0_6  = "4.0.0.6"
	aerospikeServer_4_1_0_1  = "4.1.0.1"
	aerospikeServer_4_1_0_6  = "4.1.0.6"
	aerospikeServer_4_2_0_3  = "4.2.0.3"
	aerospikeServer_4_2_0_4  = "4.2.0.4"
	aerospikeServer_4_2_0_5  = "4.2.0.5"
	aerospikeServer_4_2_0_10 = "4.2.0.10"
	aerospikeServer_4_3_0_2  = "4.3.0.2"
	aerospikeServer_4_3_0_4  = "4.3.0.4"
	aerospikeServer_4_3_0_6  = "4.3.0.6"
	aerospikeServer_4_3_0_7  = "4.3.0.7"
	aerospikeServer_4_3_0_8  = "4.3.0.8"
	aerospikeServer_4_3_0_10 = "4.3.0.10"
)

var (
	// AerospikeServerSupportedVersions holds the list of Aerospike versions
	// currently supported by the operator.
	AerospikeServerSupportedVersions = []string{
		aerospikeServer_4_0_0_4,
		aerospikeServer_4_0_0_5,
		aerospikeServer_4_0_0_6,
		aerospikeServer_4_1_0_1,
		aerospikeServer_4_1_0_6,
		aerospikeServer_4_2_0_3,
		aerospikeServer_4_2_0_4,
		aerospikeServer_4_2_0_5,
		aerospikeServer_4_2_0_10,
		aerospikeServer_4_3_0_2,
		aerospikeServer_4_3_0_4,
		aerospikeServer_4_3_0_6,
		aerospikeServer_4_3_0_7,
		aerospikeServer_4_3_0_8,
		aerospikeServer_4_3_0_10,
	}
)
