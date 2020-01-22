# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of serverless
applications and functions.

## Development and private cluster testing

### Requirements

- `podman` aliased to `docker` or `docker` (17.05 or newer)
- `podman` or `docker` is logged into a repository you can push to
- `DOCKER_REPO_OVERRIDE` points to that repository
- `envsubst`
- `bash` (4.0.0 or newer)
- `make`

### Creating the images

To test the Serverless Operator against your private Openshift cluster you first
need to push the necessary images to a publicly available location. To do that,
make sure the `DOCKER_REPO_OVERRIDE` environment variable is set to a docker
repository you can push to, for example `docker.io/markusthoemmes`. You might
need to run `docker login` to be able to push images. Now run
`make images` and all images in this repository will now be built and
pushed to your docker repository.

### Installing the system/running tests

Use the appropriate make targets or scripts in `hack`:

- `make dev`: Deploys the serverless-operator without deploying Knative Serving.
- `make install`: Scales the cluster appropriately, deploys serverless-operator
  and Knative Serving.
- `make test-e2e`: Scales, installs and runs all tests.

**Note:** Don't forget you can chain `make` targets. `make images dev` is handy
for example.

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
./hack/catalog.sh | oc apply -n openshift-marketplace -f -
```

### Create a Subscription

To install the operator, create a subscription:

```
cat <<-EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator
  generateName: serverless-operator-
  namespace: openshift-operators
spec:
  source: serverless-operator
  sourceNamespace: openshift-marketplace
  name: serverless-operator
  channel: techpreview
EOF
```
