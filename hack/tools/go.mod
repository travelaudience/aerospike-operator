module github.com/travelaudience/aerospike-operator/hack/tools

go 1.20

replace k8s.io/code-generator => k8s.io/code-generator v0.0.0-20190311093542-50b561225d70

require k8s.io/code-generator v0.0.0-20190511023357-639c964206c2

require (
	github.com/spf13/pflag v1.0.3 // indirect
	golang.org/x/tools v0.0.0-20190513233021-7d589f28aaf4 // indirect
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/klog v0.3.0 // indirect
)
