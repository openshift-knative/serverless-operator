# knative-operator

A platform-specific operator for OpenShift to be deployed with the
primary upstream knative-serving and knative-eventing operators.

## Running e2e tests

Ensure KUBECONFIG refers to a valid OpenShift cluster with
maistra/istio installed, and then run:

```
kubectl apply -f https://raw.githubusercontent.com/knative/serving-operator/master/config/300-serving-v1alpha1-knativeserving-crd.yaml
operator-sdk test local ./test/e2e/ --namespace default 
```


Eventing TODO:
----
- SMMR
  - Add namespace to SMMR
  - Wait until ServiceMesh is available
  - Delete stuff in SMMR when CR is deleted
- Network policy for eventing
- SCC related changes  
