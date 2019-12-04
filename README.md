# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of serverless
applications and functions.

## Development and private cluster testing

### Requirements

- `podman` aliased to `docker` or `docker` (17.05 or newer)
- `podman` or `docker` are logged into a repository you can push to
- `DOCKER_REPO_OVERRIDE` points to that repository
- `envsubst`
- `bash` (4.0.0 or newer)

### Creating the images

To test the Serverless Operator against your private Openshift cluster you first
need to push the necessary images to a publicly available location. To do that,
make sure the `DOCKER_REPO_OVERRIDE` environment variable is set to a docker
repository you can push to, for example `docker.io/markusthoemmes`. You might
need to run `docker login` to be able to push images. Now run
`make publish-images` and all images in this repository will now be built and
pushed to your docker repository.

### Installing the system/running tests

Use the appropriate make targets or scripts in `hack`. The system can be
installed via [`hack/install.sh`](hack/install.sh) and the entire test-suite can
be run using `make test-e2e`.

## Operator Framework

This repository contains the metadata required by the
[Operator Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager)

### Create a CatalogSource

The [catalog.sh](hack/catalog.sh) script should yield a valid `ConfigMap` and
`CatalogSource` (given you have a setup as described in the development section
above) comprised of the `ClusterServiceVersions`, `CustomResourceDefinitions`,
and package manifest in the bundle beneath [olm-catalog/](olm-catalog/). You
should apply its output in the namespace where the other `CatalogSources` live
on your cluster, e.g. `openshift-marketplace`:

```
CS_NS=$(oc get catalogsources --all-namespaces | tail -1 | awk '{print $1}')
./hack/catalog.sh | oc apply -n $CS_NS -f -
```

### Create a Subscription

To install the operator, create a subscription:

```
CS_NS=$(oc get catalogsources --all-namespaces | tail -1 | awk '{print $1}')
OPERATOR_NS=$(oc get og --all-namespaces | grep global-operators | awk '{print $1}')

cat <<-EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator-sub
  generateName: serverless-operator-
  namespace: $OPERATOR_NS
spec:
  source: serverless-operator
  sourceNamespace: $CS_NS
  name: serverless-operator
  channel: techpreview
EOF
```