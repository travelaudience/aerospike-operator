#!/bin/bash

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

set -o errexit
set -o nounset
set -o pipefail

# grab the list of files to check (ignoring vendor/ and generated code)
FILES=$(find . -type f \
    -name "*.go" \
    -not -path "./vendor/*" \
    -not -path "./pkg/client/*" \
    -not -name "zz_generated*")
# gofmt the target files
gofmt -w -s ${FILES}
# grab goimports if it is not present
command -v goimports > /dev/null || go get golang.org/x/tools/cmd/goimports
# goimports the target files
goimports -local github.com/travelaudience/aerospike-operator -w ${FILES}
