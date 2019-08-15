# Red Hat Serverless Operator

Provides a collection of API's to support deploying and serving of
serverless applications and functions.

## Operator Framework

This repository contains the metadata required by the [Operator
Lifecycle
Manager](https://github.com/operator-framework/operator-lifecycle-manager)

### Create a CatalogSource

The [catalog.sh](hack/catalog.sh) script should yield a valid
`ConfigMap` and `CatalogSource` comprised of the
`ClusterServiceVersions`, `CustomResourceDefinitions`, and package
manifest in the bundle beneath [olm-catalog/](olm-catalog/). You
should apply its output in the OLM namespace:

```
OLM_NS=$(kubectl get deploy --all-namespaces | grep olm-operator | awk '{print $1}')
./hack/catalog.sh | kubectl apply -n $OLM_NS -f -
```

### Create a Subscription

To install the operator, create a subscription:

```
OLM_NS=$(kubectl get og --all-namespaces | grep olm-operators | awk '{print $1}')
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
  sourceNamespace: $OLM_NS
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
kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.11.0/crds.yaml
kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.11.0/olm.yaml
```

Once all the pods in the `olm` namespace are running, install the
catalog source and operator as described above.

Interacting with OLM is possible using `kubectl` but the OKD console
is "friendlier". If you have docker installed, use [this
script](https://github.com/operator-framework/operator-lifecycle-manager/blob/master/scripts/run_console_local.sh)
to fire it up on <http://localhost:9000>.

