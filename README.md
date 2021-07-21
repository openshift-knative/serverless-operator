# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of serverless
applications and functions.

## Requirements

For the scripts in this repository to work properly, the following requirements apply.

- `podman` aliased to `docker` or `docker` (17.05 or newer) (only if you need/want to
build images from code)
- `bash` (4.0.0 or newer)
- `make`
- `yq` (3.4.0)

## Install an unreleased version

To install an unreleased version (either **nightly** from HEAD or the respective version
from an already cut branch), you can follow these steps. You can replace the `crc start`
step with whatever cluster you might want to use. Also make sure to check out the respective
branch of the version you want to install first.

```
$ crc start --cpus=6 --memory 16384
$ unset DOCKER_REPO_OVERRIDE # This makes sure that pre-built images are used
$ make install
```

## Development and private cluster testing

### tl;dr

If you want to quickly run the most relevant tests locally (those that are required by
the CI environment), use these commands:

```
$ crc start --cpus=6 --memory 16384
$ export DOCKER_REPO_OVERRIDE=quay.io/username
$ make images test-operator
```

### CRC-based cluster

If you want to use CRC to run tests locally, the following configuration has
been tested to work with the operator E2E tests.

```
crc start --cpus=6 --memory 16384
```

### Creating the images

To test the Serverless Operator against your private Openshift cluster you first
need to push the necessary images to a publicly available location. To do that,
make sure the `DOCKER_REPO_OVERRIDE` environment variable is set to a docker
repository you can push to, for example `quay.io/markusthoemmes`. You might
need to run `docker login` to be able to push images. Now run
`make images` and all images in this repository will now be built and
pushed to your docker repository.

### Installing the system

Use the appropriate make targets or scripts in `hack`:

- `make dev`: Deploys the serverless-operator without deploying Knative Serving, Eventing and Kafka components.
- `make install`: Scales the cluster appropriately, deploys serverless-operator, Knative Serving and Eventing.
- `make install-all`: Scales the cluster appropriately, deploys serverless-operator, Knative Serving, Eventing and Kafka.
- `make install-serving`: Scales the cluster appropriately, deploys serverless-operator and Knative Serving.
- `make install-eventing`: Scales the cluster appropriately, deploys serverless-operator and Knative Eventing.
- `make install-kafka`: Scales the cluster appropriately, deploys serverless-operator, Knative Eventing and Knative Kafka 
in `knative-eventing` namespace by default. Requires to install a Strimzi cluster with `make install-strimzi`.
- `make install-previous`: same as `make install` but deploy previous serverless-operator
  version.
- `make install-strimzi`: Install the latest Strimzi operator and a kafka cluster instance in `kafka` namespace by default.
- `make unistall-strimzi`: Uninstall the Strimzi operator and any existing kafka cluster instance. 
- `make install-mesh`: Install service mesh operator.
- `make uninstall-mesh `: Uninstall service mesh operator.
- `make install-full-mesh`: Install service mesh operator, Istio Gateway and PeerAuthentication to use Knative Serving for secure traffic.
- `make uninstall-full-mesh `: Uninstall service mesh operator, Istio Gateway and PeerAuthentication.

**Note:** Don't forget you can chain `make` targets. `make images dev` is handy
for example.

### Updating the release files

To update release files with variables stored in [project.yaml](./olm-catalog/serverless-operator/project.yaml) 
we use [generate scripts](./hack/generate) and a [templates](./templates). Update
those based on your needs, and generate the files with:

```
make release-files
```

### Running tests

#### serverless-operator tests

- `make test-unit`: Runs unit tests.
- `make test-e2e`: Scales, installs and runs E2E tests (except for Knative Kafka components).
- `make test-e2e-with-kafka`: Scales, installs and runs E2E tests (also tests Knative Kafka components).
- `make install-mesh test-e2e`: Scales, installs and runs E2E tests, including ServiceMesh integration tests
- `make test-operator`: Runs unit and E2E tests.

#### knative-serving and knative-eventing E2E tests

- `make test-upstream-upgrade`: Installs a `previous` version of Serverless and
 runs Knative Serving upgrade tests. Requirements:
     1) Running OCP cluster.
     2) Knative Serving images that the current Serverless operator depends
        on are published in CI registry. This requirement is automatically met
        when the respective branch in
        [Knative Serving](https://github.com/openshift/knative-serving) is created and its
        pre-submit CI checks run at least once.
     3) The path `${GOPATH}/src/knative.dev/serving` containing
        [Knative Serving](https://github.com/openshift/knative-serving) sources from the
        desired branch. This should be checked out before running tests.
- `make test-upstream-e2e-no-upgrade`: Installs the latest version of Serverless and
 runs Knative Serving and Knative Eventing E2E tests (without upgrades). Requirements:
     1) Running OCP cluster.
     2) Knative Serving and Knative Eventing images that the current Serverless operator depends
        on are published in CI registry. This requirement is automatically met
        when the respective branches in [Knative Serving](https://github.com/openshift/knative-serving) and
        [Knative Eventing](https://github.com/openshift/knative-eventing) are created, and their
        pre-submit CI checks run at least once.
     3) The path `${GOPATH}/src/knative.dev/serving` containing
        [Knative Serving](https://github.com/openshift/knative-serving) sources from the desired branch. 
        The path `${GOPATH}/src/knative.dev/eventing` containing
        [Knative Eventing](https://github.com/openshift/knative-eventing) sources from the desired branch. 
        This should be checked out before running tests.

#### Individual tests from knative-serving and knative-eventing

There are targets for running individual tests in both
[Knative Serving Makefile](https://github.com/openshift/knative-serving/blob/main/Makefile) and
[Knative Eventing Makefile](https://github.com/openshift/knative-eventing/blob/main/Makefile).

Example targets that can be run from the respective repositories (these targets all requires a running OCP 
cluster and pre-installed Serverless):

- `make TEST=<name_of_test> BRANCH=<ci_promotion_name> test-e2e-local`: Runs the given test using the
   latest test images that were published under the given promotion name. This doesn't require building
   any images manually.
   Example: `make BRANCH=knative-v0.15.2 TEST=TestDestroyPodInflight test-e2e-local`. 
- `make IMAGE=<name_of_image> DOCKER_REPO_OVERRIDE=<dockerhub_registry> test-image-single`: Builds an 
   image from `test/test_images/$(IMAGE)` and pushes it into the Dockerhub registry. This requires 
   ko 0.2.0 or newer.
- `make TEST=<name_of_test> DOCKER_REPO_OVERRIDE=<dockerhub_registry> test-e2e-local`: Runs the given test
   using the previously built image.

## Operator Framework

This repository contains the metadata required by the
[Operator Lifecycle Manager](https://github.com/operator-framework/operator-lifecycle-manager)

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

### Test upgrade from previous version

To test upgrade from previous version, deploy serverless operator by `make install-previous`

```
make install-previous
```

Then, you can see the installplans that the latest version has APPROVED `false`.

```
$ oc get installplan -n openshift-operators
NAME            CSV                          APPROVAL    APPROVED
install-ck6jl   serverless-operator.v1.4.1   Manual      true
install-hrzzn   serverless-operator.v1.5.0   Manual      false
```

For example, v1.5.0 is the latest version in this case. To upgrade v1.5.0, you can edit
`spec.approved` to true manually.

```
spec:
  approval: Manual
  approved: true
```

After a few minutes, operators will be upgraded automatically.

## Contributing

### Create a new version

To create a new version of the serverless-operator (usually after a release branch has
been cut), there are a few steps we have to do. These steps can be done one after the
other and do not have to be sent as one PR, to avoid clogging up the respective PR.

#### 1. Update the version metadata of the serverless-operator

The first thing we usually do is to update the version metadata of the operator. That
mostly includes bumping the version to the next desired version (i.e. 1.12 -> 1.13).
This is done by adjusting the respective settings in the `project` and `olm` part of
[`project.yaml`](./olm-catalog/serverless-operator/project.yaml). The settings to be
changed usually are `project.version`, `olm.replaces` and `olm.skipRange`.

Next, add the now outdated version of serverless-operator to the CatalogSource deployment
in [catalogsource.bash](./hack/lib/catalogsource.bash). The image to be added usually has
the following format: `registry.ci.openshift.org/openshift/openshift-serverless-$OLD_VERSION:serverless-bundle`.
Add it before the "current" image, which is `image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-bundle`.

After the changes are done, commit them and run `make generated-files`. All manifests
will now be updated accordingly. It's encouraged to commit the generated changes
separately, to ease review.

#### 2. Update the image coordinates and installation manifests

To update the image coordinates of the component you want to bump, adjust its version in
the `dependencies` section of [`project.yaml`](./olm-catalog/serverless-operator/project.yaml).

It should be a rare occasion, but between releases, the manifest files we want to pull
might have changed. If that is the case, adjust the files downloaded in
[`update-manifests.sh`](./openshift-knative-operator/hack/update-manifests.sh).

Likewise a rare occasion should be patches to the manifest files. `update-manifests.sh`
might be applying patches that can be removed in the new release or have to be adjusted.
Make sure to review them and act accordingly.

After the changes are done, commit them and run `make generated-files`. All manifests
will now be updated accordingly. It's encouraged to commit the generated changes
separately, to ease review.

#### 3. Update the Golang dependencies

The repository itself depends on the respective upstream releases. The
`openshift-knative-operator` for example is straightly build from vendoring the upstream
operator and the tests heavily rely on upstream clients, APIs and helpers.

To update the dependencies, update the `KN_VERSION` variable in
[`update-deps.sh`](./hack/update-deps.sh). Then run `./hack/update-deps.sh --upgrade` 
(like we do upstream) to pull in all the correct versions. It's encouraged to commit
the generated changes separately, to ease review.

If we're bumping our minimum supported Openshift version, bump the `OCP_VERSION` variable
in the same file and follow the same process.

### Linting

To run the linters that CI is running, you can use `make lint`.
The required linters for that are:

- [`woke`](https://github.com/get-woke/woke) to detect non-inclusive language
- [`golangci-lint`](https://golangci-lint.run/) to lint golang code
- [`shellcheck`](https://www.shellcheck.net/) to lint shell files
- [`operator-sdk`](https://sdk.operatorframework.io/docs/installation/) to lint the bundle files
- [`misspell`](https://github.com/client9/misspell) to lint typos
- [`prettier`](https://prettier.io/) to format YAML