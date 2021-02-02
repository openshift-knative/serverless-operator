Enable Istio sidecar on Knative Serving system pods
---

This documentation explains how to enable Istio sidecar on Knative Serving system pods.
The current [official documentation](https://docs.openshift.com/container-platform/4.6/serverless/networking/serverless-ossm.html) instructs how to enable sidecar for user namespaces but it does not support the sidecar injectioin for pods in system namespaces such as `knative-serving` and `knative-serving-ingress`. This documentation covers it.

Please be aware that the sidecar injection for system namespaces has a few known issues such as:

- Annotation for Istio sidecar injection may not be persisted safely. Please see [SRVKS-587]( https://issues.redhat.com/browse/SRVKS-587).
- Label `maistra.io/expose-route` for Kourier gateway may not be persisted safely.
- OpenShift ServiceMesh cannot support multi mesh since system namespaces are included in a ServiceMeshMemberRoll.

### Prerequisites

Install the [OpenShift Serverless Operator](https://docs.openshift.com/container-platform/4.6/serverless/installing_serverless/installing-openshift-serverless.html#installing-openshift-serverless) and [Knative Serving](https://docs.openshift.com/container-platform/4.6/serverless/installing_serverless/installing-knative-serving.html#installing-knative-serving).

### 1. Add namespaces into SMMR

```
$ cat <<EOF | oc apply -f -
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
  - knative-serving
  - knative-serving-ingress
  - <namespace> # A list of namespaces to be integrated with Service Mesh.
EOF
```

### 2. Add `maistra.io/expose-route` label to `3scale-kourier-gateway` deployment

```
$ oc  -n knative-serving-ingress edit deploy 3scale-kourier-gateway
```

Add the following `maistra.io/expose-route: "true"` label in `spec.template.metadata.label`.

```
  template:
    metadata:
      labels:
        app: 3scale-kourier-gateway
        maistra.io/expose-route: "true"
```

### 3. Enable istio injections

You can add annotations to system pods manually but they would not be persisted safely.

```
$ for DEPLOY in `oc get deploy -n knative-serving -o name`; do
  oc patch -n knative-serving ${DEPLOY} -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}'
done

$ for DEPLOY in `oc get deploy -n knative-serving-ingress -o name`; do
  oc patch -n knative-serving-ingress ${DEPLOY} -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}'
done
```

You can see all pods have istio sidecar.

```
$ oc get pod -n knative-serving
NAME                                    READY   STATUS    RESTARTS   AGE
activator-6d9977d8b9-4m8nx              3/3     Running   0          73s
activator-6d9977d8b9-9rz86              2/3     Running   0          72s
autoscaler-f486b855c-b8496              3/3     Running   0          72s
autoscaler-hpa-7586579894-vmrgn         3/3     Running   1          70s
autoscaler-hpa-7586579894-vtfn9         3/3     Running   0          70s
controller-6cd9c5699c-mxj8c             3/3     Running   1          70s
controller-6cd9c5699c-wwhjh             3/3     Running   0          70s
domain-mapping-6d8c6b5c69-lgg5j         3/3     Running   0          71s
domainmapping-webhook-f8c5fdb89-r9pkz   3/3     Running   1          70s
webhook-65bd8d8fbc-j4rqw                3/3     Running   0          70s
webhook-65bd8d8fbc-n66xc                3/3     Running   0          69s
```

### 4. Verify KService with injection

```
$ cat <<EOF | oc apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello-example-1
spec:
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "true"
    spec:
      containers:
      - image: docker.io/openshift/hello-openshift
        name: container
EOF
```

Please add `sidecar.istio.io/inject` in `spec.template.metadata.annotations` to inject sidecar for your KService.

```
$ URL=`oc get ksvc hello-example-1 -o jsonpath={.status.url}`

$ curl $URL
Hello OpenShift!
```
