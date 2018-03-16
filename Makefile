# the dep target fetches required dependencies
# it should be removed as soon as k8s.io/code-generator can be specified as a
# 'required' dependency in Gopkg.toml, and replaced by a call to dep ensure
# (see https://github.com/golang/dep/issues/1306)
dep:
	dep ensure
	go get -d k8s.io/code-generator/...
	go get -d k8s.io/apimachinery/pkg/apimachinery/registered

fmt:
	./hack/update-fmt.sh

gen: export CODEGEN_PKG=../../../k8s.io/code-generator
gen:
	./hack/update-codegen.sh

run: KUBECONFIG?=$(HOME)/.kube/config
run:
	go run cmd/operator/main.go -debug -kubeconfig=$(KUBECONFIG)
