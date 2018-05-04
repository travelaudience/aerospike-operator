# Building `aerospike-operator`

## Pre-requisites

* Kubernetes 1.9+.
* Go 1.10+
* `dep`
* `make`
* [`skaffold`](https://github.com/GoogleContainerTools/skaffold)

## Grabbing the source code

```
$ git clone git@github.com/travelaudience/aerospike-operator.git
```

## Installing dependencies

To install all the required dependencies one may run

```
$ make dep
```

This will fetch project dependencies using `dep` as specified in `Gopkg.lock` as
well as some packages required for code generation. The latter are fetched using
`go get`, but should be moved to `Gopkg.toml` as soon as
[this issue](https://github.com/golang/dep/issues/1306) is fixed.

## Generating code

Part of working with `CustomResourceDefinition`s involves generating code using
[`k8s.io/code-generator`](https://github.com/kubernetes/code-generator). The
following files and directories correspond to generated code:

```
pkg
├── apis
│   └── aerospike
│       └── v1alpha1
│           └── zz_generated.deepcopy.go (GENERATED)
└── client (GENERATED)
```

The code generation step should be run whenever a modification to `pkg/apis` is
made. To run code generation, one can use

```
$ make gen
```

## Running `aerospike-operator`

`aerospike-operator` must run inside a Kubernetes cluster, and `skaffold` is
used in order to ease the deployment of local versions. To deploy
`aerospike-operator` to the Minkube cluster targeted by the current context one
may run

```
$ PROFILE=minikube make dev
```

or, if using GKE,

```
$ PROFILE=gke make dev
```

This will create or update everything that is needed for `aerospike-operator`
and the test suite to run, and will create an `aerospike-operator` pod inside
the `aerospike-operator` namespace. Since the pods runs `go run`, it may take
a couple of minutes for the logs to start being presented. Once one sees

```
...
[aerospike-operator] time="2018-04-26T09:55:11Z" level=debug msg="starting workers"
[aerospike-operator] time="2018-04-26T09:55:11Z" level=debug msg="started workers"
```

one may proceed to running the test suite.

## Running the test suite

Once `aerospike-operator` is deployed using the steps describe above one may run
the test suite against the cluster by running

```
$ make test.e2e
```

This will run the test suite from within the `aerospike-operator` pod and
display the results once they are ready:

```
...
Ran 14 of 14 Specs in 916.889 seconds
SUCCESS! -- 14 Passed | 0 Failed | 0 Flaked | 0 Pending | 0 Skipped --- PASS: TestE2E (916.89s)
...
```
