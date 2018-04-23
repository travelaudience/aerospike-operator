# the dep target fetches required dependencies
# it should be removed as soon as k8s.io/code-generator can be specified as a
# 'required' dependency in Gopkg.toml, and replaced by a call to dep ensure
# (see https://github.com/golang/dep/issues/1306)
.PHONY: dep
dep: KUBERNETES_VERSION=1.9.6
dep: KUBERNETES_CODE_GENERATOR_PKG=k8s.io/code-generator
dep: KUBERNETES_APIMACHINERY_PKG=k8s.io/apimachinery
dep:
	dep ensure
	go get -d $(KUBERNETES_CODE_GENERATOR_PKG)/...
	cd $(GOPATH)/src/$(KUBERNETES_CODE_GENERATOR_PKG) && \
		git fetch origin && \
		git checkout -f kubernetes-$(KUBERNETES_VERSION)
	go get -d $(KUBERNETES_APIMACHINERY_PKG)/...
	cd $(GOPATH)/src/$(KUBERNETES_APIMACHINERY_PKG) && \
		git fetch origin && \
		git checkout -f kubernetes-$(KUBERNETES_VERSION)

.PHONY: docker.operator
docker.operator: TAG?=$(shell git describe --dirty)
docker.operator: IMG?=quay.io/travelaudience/aerospike-operator
docker.operator:
	docker build -t $(IMG):$(TAG) -f ./Dockerfile .

.PHONY: docker.tools
docker.tools: TAG?=$(shell git describe --dirty)
docker.tools: IMG?=quay.io/travelaudience/aerospike-operator-tools
docker.tools:
	docker build -t $(IMG):$(TAG) -f ./Dockerfile.tools .

.PHONY: fmt
fmt:
	./hack/update-fmt.sh

.PHONY: gen
gen: export CODEGEN_PKG=../../../k8s.io/code-generator
gen:
	./hack/update-codegen.sh

.PHONY: run
run: POD_NAMESPACE?=aerospike-operator
run: KUBECONFIG?=$(HOME)/.kube/config
run:
	POD_NAMESPACE=$(POD_NAMESPACE) go run cmd/operator/main.go \
	    -debug \
	    -kubeconfig=$(KUBECONFIG)

.PHONY: test.unit
test.unit:
	go test -v ./cmd/... ./pkg/...

.PHONY: test.e2e
ifneq ($(LOCAL),1)
test.e2e: TAG?=$(shell git describe --dirty)
test.e2e: IMG?=quay.io/travelaudience/aerospike-operator
test.e2e: EXTRA_FLAGS:=-operator-image=$(IMG):$(TAG)
endif
ifdef NODE_ADDRESS
test.e2e: EXTRA_FLAGS+=-node-address=$(NODE_ADDRESS)
endif
test.e2e: FLAKE_ATTEMPTS?=3
test.e2e: FOCUS?=
test.e2e: KUBECONFIG?=$(HOME)/.kube/config
test.e2e: TIMEOUT?=1200s
test.e2e:
	-kubectl apply -f docs/examples/00-prereqs.yml
	go test -v -timeout=$(TIMEOUT) ./test/e2e -kubeconfig=$(KUBECONFIG) -ginkgo.flakeAttempts=$(FLAKE_ATTEMPTS) -ginkgo.focus=$(FOCUS) $(EXTRA_FLAGS)
