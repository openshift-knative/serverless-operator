# Test serverless-operator with Istio sidecar injection

To install service mesh operator, run `make install-mesh`

```
make install-mesh
```

and create a `ServiceMeshControlPlane`.

```
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: basic
  namespace: istio-system
spec:
  version: v2.0
```

Then, add your namespace to `ServiceMeshMemberRoll`

```
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
    - $NAMESPACE_YOU_WANT_TO_ADD
    # Add namespace you want to include mesh.
```

and add `NetworkPolicy` in your namespace.

```
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-system-namespace
  namespace: $NAMESPACE_YOU_WANT_TO_ADD
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          knative.openshift.io/part-of: "openshift-serverless"
  podSelector: {}
  policyTypes:
  - Ingress
```

Then, create Knative Service with `sidecar.istio.io/inject: "true"` label in your namespace,
which is one of the namespaces in the `ServiceMeshMemberRoll`.

```sh
cat <<EOF | oc apply -n $NAMESPACE_YOU_WANT_TO_ADD -f -
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: hello-example
spec:
  template:
    metadata:
      name: hello-example-1
      labels:
        sidecar.istio.io/inject: "true"
    spec:
      containers:
      - image: gcr.io/knative-samples/helloworld-go
        name: user-container
EOF
```

To uninstall service mesh operator, run `make uninstall-mesh`.

```
make uninstall-mesh
```
