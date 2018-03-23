# the dep target fetches required dependencies
# it should be removed as soon as k8s.io/code-generator can be specified as a
# 'required' dependency in Gopkg.toml, and replaced by a call to dep ensure
# (see https://github.com/golang/dep/issues/1306)
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

fmt:
	./hack/update-fmt.sh

gen: export CODEGEN_PKG=../../../k8s.io/code-generator
gen:
	./hack/update-codegen.sh

run: KUBECONFIG?=$(HOME)/.kube/config
run:
	go run cmd/operator/main.go -debug -kubeconfig=$(KUBECONFIG)

test:
	go test ./...
