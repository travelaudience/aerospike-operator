# VERSION holds the current version of aerospike-operator.
VERSION?=0.10.1

build: BIN?=operator
build: OUT?=bin/aerospike-operator
build: GOOS?=linux
build: GOARCH?=amd64
build: gen
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build \
		-v \
		-ldflags="-d -s -w -X github.com/travelaudience/aerospike-operator/pkg/versioning.OperatorVersion=$(VERSION)" \
		-tags=netgo \
		-installsuffix=netgo \
		-o=$(OUT) ./cmd/$(BIN)/main.go

.PHONY: skaffold
skaffold: NAMESPACE := aerospike-operator
skaffold: PROFILE ?= minikube
skaffold: PROJECT_ID ?= aerospike-operator
skaffold: TARGET := operator
skaffold:
	@NAMESPACE="$(NAMESPACE)" \
	PROFILE="$(PROFILE)" \
	PROJECT_ID="$(PROJECT_ID)" \
	TARGET="$(TARGET)" \
	./hack/skaffold/skaffold.sh

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

.PHONY: openapi
openapi: CONTAINER_NAME?=aerospike-operator-openapi
openapi: OPEN?=0
openapi: PORT?=8080
openapi:
	@./hack/update-openapi.sh
ifeq ($(OPEN),1)
	@docker rm -f $(CONTAINER_NAME) || true && docker run \
		-d \
		-e SWAGGER_JSON=/swagger.json \
		--name $(CONTAINER_NAME) \
	    -p $(PORT):8080 \
	    -v $(PWD)/docs/design/swagger.json:/swagger.json \
	    swaggerapi/swagger-ui:3.17.4
	@open http://localhost:$(PORT)/
endif

.PHONY: gen
gen:
	@./hack/update-codegen.sh

.PHONY: test.unit
test.unit:
	go test -v ./cmd/... ./pkg/...

.PHONY: test.e2e.build
test.e2e.build:
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go test -c \
		-v \
		-installsuffix=netgo \
		-tags=netgo \
		-ldflags='-d -s -w' \
		-o=bin/aerospike-operator-e2e test/e2e/*.go

.PHONY: test.e2e
test.e2e: FOCUS ?=
test.e2e: GCS_BUCKET_NAME ?= aerospike-operator
test.e2e: NAMESPACE := aerospike-operator-e2e
test.e2e: PROFILE ?= minikube
test.e2e: PROJECT_ID ?= aerospike-operator
test.e2e: SKIP ?=
test.e2e: STORAGE_ADMIN_KEY_JSON_FILE ?= ./key.json
test.e2e: TARGET := e2e
test.e2e:
	@kubectl delete namespace --ignore-not-found $(NAMESPACE)
	@FOCUS=$(FOCUS) \
	NAMESPACE=$(NAMESPACE) \
	PROFILE=$(PROFILE) \
	PROJECT_ID=$(PROJECT_ID) \
	SKIP=$(SKIP) \
	STORAGE_ADMIN_KEY_JSON_FILE=$(STORAGE_ADMIN_KEY_JSON_FILE) \
	TARGET=$(TARGET) \
	./hack/skaffold/skaffold.sh
