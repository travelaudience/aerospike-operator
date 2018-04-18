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

# grab the path to the repo
SCRIPT_ROOT="$(dirname $(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd))"
# grab the path to the codegen package
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
# run codegen
${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/travelaudience/aerospike-operator/pkg/client \
  github.com/travelaudience/aerospike-operator/pkg/apis \
  aerospike:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
