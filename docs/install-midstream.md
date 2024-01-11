# Installing and upgrading OpenShift Serverless midstream

Navigate to resources directory:

```shell
cd resources-install-midstream
```

## Installation

Create the `openshift-serverless` namespace:

```shell
oc apply -f 100-oss-namespace.yaml
```

Create an `OperatorGroup`:

```shell
oc apply -f 200-oss-operator-group.yaml
```

### Create a CatalogSource for each version

For each branch of OSS you wish to install or upgrade, install a CatalogSource like the following:

```yaml
# CatalogSource for 1.24
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  # Replace the version with whatever version you want to install
  name: serverless-operator-v1-24-0
  namespace: openshift-marketplace
spec:
  displayName: Serverless Operator
  # Replace the version with whatever version you want to install
  image: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0:serverless-index
  publisher: Red Hat
  sourceType: grpc
---
# CatalogSource for 1.25
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  # Replace the version with whatever version you want to install
  name: serverless-operator-v1-25-0
  namespace: openshift-marketplace
spec:
  displayName: Serverless Operator
  # Replace the version with whatever version you want to install
  image: registry.ci.openshift.org/knative/openshift-serverless-v1.25.0:serverless-index
  publisher: Red Hat
  sourceType: grpc
---
# CatalogSource for main (in development, unstable)
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  # Replace the version with whatever version you want to install
  name: serverless-operator-nightly
  namespace: openshift-marketplace
spec:
  displayName: Serverless Operator
  # Replace the version with whatever version you want to install
  image: registry.ci.openshift.org/knative/serverless-index:main
  publisher: Red Hat
  sourceType: grpc
```

__CatalogSource images aren't bound to a specific `patch` version, every commit to a specific branch will actually
change the `vx.y.0` image (they are streams of major versions).__

### Installation

Apply the following subscription referencing the specific catalog source you want to install:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator
  namespace: openshift-serverless
spec:
  channel: stable
  name: serverless-operator
  # Change the CatalogSource name
  source: serverless-operator-v1-24-0
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
```

### Upgrade

Apply the following subscription referencing the specific catalog source you want to upgrade to:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator
  namespace: openshift-serverless
spec:
  channel: stable
  name: serverless-operator
  # Change the CatalogSource name
  source: serverless-operator-v1-25-0
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
```

### Downgrade

OLM **does not** support downgrades, so referencing a previous version in a Subscription isn't supported, so we need a few
steps to work around the limitation.

1. Delete the `Subscription`

  ```shell
  oc delete subscriptions.operators.coreos.com -n openshift-serverless serverless-operator
  ```

2. Delete the `ClusterServiceVersion` associated with the specific release you have installed:

  ```shell
  oc delete csv -n openshift-serverless serverless-operator.v1.25.0
  ```

This step deletes the pods in `openshift-serverless`, these aren't critical since they provide a way to install Knative
components, that means that until step 3 is complete, installation and configuration functionality on KnativeServing,
KnativeEventing, and KnativeKafka resources won't work but Knative components will be up and running as previously.

3. Update and apply the subscription to use a `CatalogSource` for a previous version:

  ```yaml
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: serverless-operator
    namespace: openshift-serverless
  spec:
    channel: stable
    name: serverless-operator
    # Change the CatalogSource name
    source: serverless-operator-v1-24-0
    sourceNamespace: openshift-marketplace
    installPlanApproval: Automatic
  ```
