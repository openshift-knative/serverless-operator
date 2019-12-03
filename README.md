# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of
serverless applications and functions.

## Local/Private cluster testing

To test the Serverless Operator against your private Openshift cluster
you first need to push the necessary images to a publicly available location.
To do that, make sure the `DOCKER_REPO_OVERRIDE` environment variable is set
to a docker repository you can push to, for example `docker.io/markusthoemmes`.
You might need to run `docker login` to be able to push images. Now run
`make publish-images` and all images in this repository will now be built and
pushed to your docker repository.

After that is done, all the scripts in the `hack` directory are at your disposal
to install and test the system. `make test-e2e` in particular runs the entirety
of the end-to-end tests against the system.

## Operator Framework

This repository contains the metadata required by the [Operator
Lifecycle
Manager](https://github.com/operator-framework/operator-lifecycle-manager)

### Create a CatalogSource

The [catalog.sh](hack/catalog.sh) script should yield a valid
`ConfigMap` and `CatalogSource` comprised of the
`ClusterServiceVersions`, `CustomResourceDefinitions`, and package
manifest in the bundle beneath [olm-catalog/](olm-catalog/). You
should apply its output in the namespace where the other
`CatalogSources` live on your cluster,
e.g. `openshift-marketplace`:

```
CS_NS=$(kubectl get catalogsources --all-namespaces | tail -1 | awk '{print $1}')
./hack/catalog.sh | kubectl apply -n $CS_NS -f -
```

### Create a Subscription

To install the operator, create a subscription:

```
CS_NS=$(kubectl get catalogsources --all-namespaces | tail -1 | awk '{print $1}')
OPERATOR_NS=$(kubectl get og --all-namespaces | grep global-operators | awk '{print $1}')

cat <<-EOF | kubectl apply -f -
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

## Using OLM on Minikube

You can test the operator using
[minikube](https://kubernetes.io/docs/setup/minikube/) after
installing OLM on it:

```
minikube start
kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.12.0/crds.yaml
kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.12.0/olm.yaml
```

Once all the pods in the `olm` namespace are running, install the
catalog source and operator as described above.

Interacting with OLM is possible using `kubectl` but the OKD console
is "friendlier". If you have docker installed, use [this
script](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/scripts/run_console_local.sh)
to fire it up on <http://localhost:9000>.

