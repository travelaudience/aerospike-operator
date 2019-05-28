module github.com/travelaudience/aerospike-operator

go 1.12

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190516230258-a675ac48af67
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190516231611-bf6753f2aa24
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190516230822-f89599b3f645
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190516230509-ae8359b20417+incompatible
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.2
)

require (
	cloud.google.com/go v0.26.0
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/PuerkitoBio/purell v1.1.0 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/aerospike/aerospike-client-go v1.35.2
	github.com/appscode/kutil v0.0.0-20180809044522-b50ebf9375cc
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/etcd v3.3.8+incompatible // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20180511133405-39ca1b05acc7 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/emicklei/go-restful v2.8.0+incompatible // indirect
	github.com/evanphx/json-patch v3.0.0+incompatible // indirect
	github.com/go-openapi/jsonpointer v0.0.0-20180322222829-3a0015ad55fa // indirect
	github.com/go-openapi/jsonreference v0.0.0-20180322222742-3fb327e6747d // indirect
	github.com/go-openapi/spec v0.0.0-20180710175419-bce47c9386f9
	github.com/go-openapi/swag v0.0.0-20180703152219-2b0bd4f193d0 // indirect
	github.com/gogo/protobuf v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20180513044358-24b0969c4cb7 // indirect
	github.com/google/btree v0.0.0-20180124185431-e89373fe6b4a // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/google/martian v2.1.0+incompatible // indirect
	github.com/googleapis/gax-go v2.0.0+incompatible // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.0 // indirect
	github.com/hashicorp/golang-lru v0.0.0-20180201235237-0fb14efe8c47 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/json-iterator/go v0.0.0-20180701071628-ab8a2e0c74be // indirect
	github.com/mailru/easyjson v0.0.0-20180717111219-efc7eb8984d6 // indirect
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742 // indirect
	github.com/munnerz/goautoneg v0.0.0-20190414153302-2ae31c8b6b30 // indirect
	github.com/onsi/ginkgo v1.5.0
	github.com/onsi/gomega v1.4.0
	github.com/pborman/uuid v0.0.0-20170612153648-e790cca94e6c // indirect
	github.com/sirupsen/logrus v1.0.5
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/stretchr/testify v1.3.0
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/ugorji/go v1.1.1 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/yuin/gopher-lua v0.0.0-20180630135845-46796da1b0b4 // indirect
	go.etcd.io/bbolt v1.3.2 // indirect
	go.opencensus.io v0.13.0 // indirect
	golang.org/x/crypto v0.0.0-20180621125126-a49355c7e3f8 // indirect
	golang.org/x/net v0.0.0-20181220203305-927f97764cc3
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sync v0.0.0-20190423024810-112230192c58 // indirect
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2 // indirect
	google.golang.org/api v0.0.0-20180702000508-e0f3bfad2532
	gopkg.in/airbrake/gobrake.v2 v2.0.9 // indirect
	gopkg.in/gemnasium/logrus-airbrake-hook.v2 v2.1.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0-20170531160350-a96e63847dc3 // indirect
	k8s.io/api v0.0.0-20180904230853-4e7be11eab3f
	k8s.io/apiextensions-apiserver v0.0.0-20180910084140-b05d9bb7cc74
	k8s.io/apimachinery v0.0.0-20190515023456-b74e4c97951f
	k8s.io/apiserver v0.0.0-20180910083620-386115dd78fd // indirect
	k8s.io/client-go v0.0.0-20180910083459-2cefa64ff137
	k8s.io/component-base v0.0.0-20190515024022-2354f2393ad4 // indirect
	k8s.io/kube-openapi v0.0.0-20190510232812-a01b7d5d6c22
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20190520173318-324c5df7d3f0 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
