#!/bin/sh

# Copyright 2018 The aerospike-operator Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# run /aerospike-operator-e2e with the requested parameters
/aerospike-operator-e2e \
	-ginkgo.flakeAttempts="${FLAKE_ATTEMPTS}" \
	-ginkgo.focus="${FOCUS}" \
    -ginkgo.progress \
    -ginkgo.v \
	-gcs-bucket-name="${GCS_BUCKET_NAME}" \
	-gcs-secret-name="${GCS_SECRET_NAME}" \
	-test.timeout="${TIMEOUT}" \
    -test.v
