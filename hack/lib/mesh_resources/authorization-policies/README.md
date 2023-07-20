The resources in this folder are based on https://github.com/openshift-knative/knative-istio-authz-chart.
`setup` can be copied 1:1, the other ones are generated using the helm generator:

```bash
helm template oci://quay.io/pierdipi/knative-istio-authz-onboarding --version 0.1.0 --set "name=tenant-1" --set "namespaces={tenant-1}" > helm-tenant-1.yaml

helm template oci://quay.io/pierdipi/knative-istio-authz-onboarding --version 0.1.0 --set "name=tenant-2" --set "namespaces={tenant-2}" > helm-tenant-2.yaml

helm template oci://quay.io/pierdipi/knative-istio-authz-onboarding --version 0.1.0 --set "name=serving-tests" --set "namespaces={serving-tests}" > helm-serving-tests.yaml
```


