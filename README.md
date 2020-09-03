# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of serverless
applications and functions.

## Development and private cluster testing

### tl;dr

If you want to quickly run the most relevant tests locally (those that are required by
the CI environment), use these commands:

```
$ crc start --cpus=6 --memory 16384
$ make images test-operator
```

### Requirements

- `podman` aliased to `docker` or `docker` (17.05 or newer)
- `podman` or `docker` is logged into a repository you can push to
- `DOCKER_REPO_OVERRIDE` points to that repository
- `envsubst`
- `bash` (4.0.0 or newer)
- `make`

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
repository you can push to, for example `docker.io/markusthoemmes`. You might
need to run `docker login` to be able to push images. Now run
`make images` and all images in this repository will now be built and
pushed to your docker repository.

### Installing the system

Use the appropriate make targets or scripts in `hack`:

- `make dev`: Deploys the serverless-operator without deploying Knative Serving and Eventing.
- `make install`: Scales the cluster appropriately, deploys serverless-operator, Knative Serving and Eventing.
- `make install-previous`: same as `make install` but deploy previous serverless-operator
  version.
- `make install-mesh`: Install service mesh operator and enable sidecar injections.
- `make uninstall-mesh `: Uninstall service mesh operator and disable sidecar injection.

**Note:** Don't forget you can chain `make` targets. `make images dev` is handy
for example.

### Running tests

#### serverless-operator tests

- `make test-unit`: Runs unit tests.
- `make test-e2e`: Scales, installs and runs E2E tests.
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
[Knative Serving Makefile](https://github.com/openshift/knative-serving/blob/master/Makefile) and
[Knative Eventing Makefile](https://github.com/openshift/knative-eventing/blob/master/Makefile).

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

### Test serverless-operator with Istio sidecar injection

To install service mesh operator, run `make install-mesh`

```
make install-mesh
```

Then, create Knative Service with `sidecar.istio.io/inject: "true"` annotation in `default` namespace,
which is one of the namespaces in the `ServiceMeshMemberRoll`.

```sh
cat <<EOF | oc apply -n default -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello-example
spec:
  template:
    metadata:
      name: hello-example-1
      annotations:
        sidecar.istio.io/inject: "true"
    spec:
      containers:
      - image: gcr.io/knative-samples/helloworld-go
        name: user-container
  traffic:
  - latestRevision: true
    percent: 100
EOF
```

`make install-mesh` creates `ServiceMeshMemberRoll` with `default` namespaces.
If you want to add more namespaces, please modify it.

```
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
    - default
    # Add namespace you want to include mesh.
```

And add `NetworkPolicy` in your namespace.

```
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-serving-system-namespace
  namespace: $NAMESPACE_YOU_WANT_TO_ADD
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          serving.knative.openshift.io/system-namespace: "true"
  podSelector: {}
  policyTypes:
  - Ingress
```

To uninstall service mesh operator, run `make uninstall-mesh`.

```
make uninstall-mesh
```
