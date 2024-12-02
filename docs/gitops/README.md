## OpenShift Serverless with GitOps

### Install GitOps operator

```shell
# Create OpenShift GitOps subscription
oc apply -f docs/gitops/gitops-subscription.yaml

# Wait for GitOps CSV to succeed
# ...
oc get csv -n openshift-gitops -w
```
### Install Serverless Catalog Source

If you are developing the Serverless operator, you can use the local catalog source as follows.

```shell
# Install the local catalog index
export DOCKER_REPO_OVERRIDE=...
make image; make install-with-argo-cd

# Modify the 100-subscriptions.yaml to use the local catalog source
# add the CatalogSource
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: serverless-operator
  namespace: openshift-marketplace
  annotations:
    argocd.argoproj.io/sync-wave: "2"
spec:
  displayName: Serverless Operator
  image: image-registry.openshift-image-registry.svc:5000/openshift-marketplace/serverless-index:latest
  publisher: Red Hat
  sourceType: grpc
 ---

# Modify the Susbcription to point to the right source
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
...
  source: "serverless-operator"
...
````


### Creating Argo CD application

```shell
# Create Argo CD application
oc apply -f docs/gitops/application.yaml

# Grant admin permission to openshift gitops controller in knative-eventing
oc adm policy add-role-to-user admin system:serviceaccount:openshift-gitops:openshift-gitops-argocd-application-controller -n knative-eventing
oc adm policy add-role-to-user admin system:serviceaccount:openshift-gitops:openshift-gitops-argocd-application-controller -n knative-serving
oc adm policy add-role-to-user admin system:serviceaccount:openshift-gitops:openshift-gitops-argocd-application-controller -n knative-serving-ingress
oc adm policy add-role-to-user admin system:serviceaccount:openshift-gitops:openshift-gitops-argocd-application-controller -n default
# Access the Argo CD UI by navigating to the openshift-gitops-server route
oc get routes -n openshift-gitops openshift-gitops-server

```

- Click Sync on openshift-serverless application
- then select only namespaces and the openshift-serverless subscription
- Once Sync is OK, sync everything

> NOTE: It may be required to _retry_ the synchronization, if resources are unhealthy

### Verify installation

```shell
# Verify that KnativeEventing, KnativeServing are ready and pods are present
oc get knativeeventing -n knative-eventing
oc get pods -n knative-eventing
oc get knativeserving -n knative-serving
oc get pods -n knative-serving
```

### Reproduce SRVCOM-2200

- Delete Serverless operator pods
  ```shell
  oc delete pods -n openshift-serverless --all
  ```
- Verify that:
  - Argo CD UI still reports `Sync OK`
  - Argo CD application still reports `Synced` with `oc get application -n openshift-gitops openshift-serverless`
