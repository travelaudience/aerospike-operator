# Building `aerospike-operator`

## Pre-requisites

* Go 1.10+
* `dep`

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
$ make codegen
```

## Running locally

To run `aerospike-operator` locally against a Kubernetes cluster, one must have
a working `kubeconfig` pointing to the cluster. Assuming the `kubeconfig` file
is located in `${HOME}/.kube/config`, one can simply run

```
$ make run
```

To specify a custom `kubeconfig`, one may run

```
$ KUBECONFIG=/path/to/kubeconfig make run
```
