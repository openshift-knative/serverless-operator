# metadata-webhook

Knative `metadata-webhook` defines a few webhook to inject metadata.

## Prerequisites

metadata-webhook requires Knative Serving.
Please install [Knative Serving](https://knative.dev/docs/install/).

## Get started

### Deploy metadata-webhook

```
$ kubectl apply -f ./config

$ kubectl get pod -n serving-tests -w
NAME                       READY   STATUS        RESTARTS   AGE
webhook-69f8fc4b4d-qp2gg   1/1     Running       0          40s
```

### Create a Knative Service

```
$ cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello
  namespace: serving-tests
spec:
  template:
    metadata:
      name: hello-1
    spec:
      containers:
      - image: docker.io/openshift/hello-openshift
        name: container
EOF
```

### Deployment has istio annotations

```
$ kubectl get -n serving-tests deploy hello-1-deployment -o=jsonpath='{.spec.template.metadata.annotations}' \
             | grep '"sidecar.istio.io/rewriteAppHTTPProbers":"true"'
$ kubectl get -n serving-tests deploy hello-1-deployment -o=jsonpath='{.spec.template.metadata.labels}' \
             | grep '"sidecar.istio.io/inject":"true"'
```
