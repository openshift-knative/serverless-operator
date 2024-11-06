Serverless must-gather
=================

`must-gather` is a tool built on top of [OpenShift
must-gather](https://github.com/openshift/must-gather) that expands
its capabilities to gather Serverless debug information.

### Usage
```sh
oc adm must-gather --image=quay.io/openshift-knative/must-gather
```

In order to get data about other parts of the cluster (not specific to
Serverless) you should run `oc adm must-gather` (without passing a
custom image). Run `oc adm must-gather -h` to see more options.

### Development
You can build the image locally using the Dockerfile included.

A `Makefile` is also provided. To use it, you must pass a repository
via the command-line using the variable `IMAGE_NAME`. You can also
specify the registry using the variable `IMAGE_REGISTRY` (default is
[quay.io](https://quay.io)) and the tag via `IMAGE_TAG` (default is
`latest`).

For example, to build and push:
```sh
make IMAGE_NAME=openshift-knative/must-gather
```
