# VERSION holds the current version of aerospike-operator.
VERSION?=0.7.0-dev

build: BIN?=operator
build: OUT?=bin/aerospike-operator
run: GOOS?=linux
run: GOARCH?=amd64
build: dep gen
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-a \
		-v \
		-ldflags="-d -s -w -X github.com/travelaudience/aerospike-operator/pkg/versioning.OperatorVersion=$(VERSION)" \
		-tags=netgo \
		-installsuffix=netgo \
		-o=$(OUT) ./cmd/$(BIN)/main.go

# the dep target fetches required dependencies
# it should be removed as soon as k8s.io/code-generator can be specified as a
# 'required' dependency in Gopkg.toml, and replaced by a call to dep ensure
# (see https://github.com/golang/dep/issues/1306)
.PHONY: dep
dep: KUBERNETES_VERSION=1.10.5
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

.PHONY: run
run: GOOS?=linux
run: GOARCH?=amd64
run: PROFILE?=minikube
run:
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-v \
		-ldflags="-d -s -w -X github.com/travelaudience/aerospike-operator/pkg/versioning.OperatorVersion=$(VERSION)" \
		-tags=netgo \
		-installsuffix=netgo \
		-o=bin/aerospike-operator ./cmd/operator/main.go
	@skaffold run -f skaffold.operator.yaml -p $(PROFILE)

.PHONY: docker.operator
docker.operator: TAG?=$(VERSION)
docker.operator: IMG?=quay.io/travelaudience/aerospike-operator
docker.operator:
	docker build -t $(IMG):$(TAG) -f ./Dockerfile .

.PHONY: docker.tools
docker.tools: TAG?=$(VERSION)
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

.PHONY: test.unit
test.unit:
	go test -v ./cmd/... ./pkg/...

.PHONY: test.e2e
test.e2e: PROFILE?=minikube
test.e2e: FLAKE_ATTEMPTS?=3
test.e2e: FOCUS?=
test.e2e: GCS_BUCKET_NAME?=
test.e2e: GCS_SECRET_NAME?=
test.e2e: TIMEOUT?=3600s
test.e2e: NAMESPACE?=aerospike-operator-e2e
test.e2e:
	@kubectl delete namespace --ignore-not-found $(NAMESPACE)
	@FLAKE_ATTEMPTS=$(FLAKE_ATTEMPTS) FOCUS=$(FOCUS) GCS_BUCKET_NAME=$(GCS_BUCKET_NAME) GCS_SECRET_NAME=$(GCS_SECRET_NAME) TIMEOUT=$(TIMEOUT) \
		envsubst < hack/run-e2e.sh > bin/aerospike-operator-e2e.sh
	@chmod +x bin/aerospike-operator-e2e.sh
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c \
		-v \
		-tags=netgo \
		-installsuffix=netgo \
		-o=bin/aerospike-operator-e2e test/e2e/*.go
	@skaffold run -f skaffold.e2e.yaml -p $(PROFILE)

.PHONY: test.e2e.logs
test.e2e.logs: NAMESPACE?=aerospike-operator-e2e
test.e2e.logs: POD?=aerospike-operator-e2e
test.e2e.logs:
	@kubectl -n $(NAMESPACE) logs -f $(POD)
