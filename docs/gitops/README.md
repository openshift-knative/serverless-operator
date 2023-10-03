## OpenShift Serverless with GitOps

### Install GitOps operator

```shell
# Create OpenShift GitOps subscription
oc apply -f docs/gitops/gitops-subscription.yaml

# Wait for GitOps CSV to succeed
# ...
oc get csv -n openshift-gitops -w
```

### Creating Argo CD application

```shell
# Create Argo CD application
oc apply -f docs/gitops/application.yaml

# Grant admin permission to openshift gitops controller in knative-eventing
oc adm policy add-role-to-user admin system:serviceaccount:openshift-gitops:openshift-gitops-argocd-application-controller -n knative-eventing

# Access the Argo CD UI by navigating to the openshift-gitops-server route
oc get routes -n openshift-gitops openshift-gitops-server

```

- Click Sync on openshift-serverless application
- then select only namespaces and the openshift-serverless subscription
- Once Sync is OK, sync everything

> NOTE: It may be required to _retry_ the synchronization, if resources are unhealthy

### Verify installation

```shell
# Verify that KnativeEventing is ready and pods are present
oc get knativeeventing -n knative-eventing
oc get pods -n knative-eventing
```

### Reproduce SRVCOM-2200

- Delete Serverless operator pods
  ```shell
  oc delete pods -n openshift-serverless --all
  ```
- Verify that:
  - Argo CD UI still reports `Sync OK`
  - Argo CD application still reports `Synced` with `oc get application -n openshift-gitops openshift-serverless`
