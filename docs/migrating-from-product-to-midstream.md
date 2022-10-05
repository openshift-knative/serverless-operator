Migration from OpenShift Serverless operator installed using product catalog to the same version using the midstream catalog:

Notes: 
- This document is Knative Eventing and Kafka specific.
- Instructions here are for 1.24 of the operator. While procedure is same for other versions, the version numbers and job names will be different.

```shell

# 1. delete Product subscription
oc delete subscriptions.operators.coreos.com -n openshift-serverless serverless-operator

# 2. delete ClusterServiceVersion
# This step deletes the pods in `openshift-serverless`, these aren't critical since they provide a way to install Knative
# components, that means that until last step is complete, installation and configuration functionality on KnativeServing,
# KnativeEventing, and KnativeKafka resources won't work but Knative components will be up and running as previously.
oc delete csv -n openshift-serverless serverless-operator.v1.24.0

# 3. delete some jobs that the new operator installation cannot modify (immutable image)
oc delete job -n knative-eventing kafka-controller-post-install-1.24.0              --ignore-not-found
oc delete job -n knative-eventing knative-kafka-storage-version-migrator-1.24.0     --ignore-not-found
oc delete job -n knative-eventing storage-version-migration-eventing-eventing-1.3.2 --ignore-not-found

# 4. create catalogSource for OpenShift Serverless midstream 1.24
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: serverless-operator-v1-24-0
  namespace: openshift-marketplace
spec:
  displayName: Serverless Operator
  image: registry.ci.openshift.org/knative/openshift-serverless-v1.24.0:serverless-index
  publisher: Red Hat
  sourceType: grpc
EOF

# wait until catalog source is ready
oc wait catalogsources -n openshift-marketplace serverless-operator-v1-24-0 --for=jsonpath='{.status.connectionState.lastObservedState}'="READY" --timeout=5m

# 5. Create a subscription to use a `CatalogSource` for a midstream version:
cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: serverless-operator
  namespace: openshift-serverless
spec:
  channel: stable
  name: serverless-operator
  source: serverless-operator-v1-24-0
  sourceNamespace: openshift-marketplace
  installPlanApproval: Automatic
EOF

# 6. Verify that images are from registry.ci.openshift.org/ and not from production registry and they are ready
# TODO: this step is only for human eyes. It doesn't fail if things are not good or wait until they're good
# These commands should produce no output
oc get pods -n openshift-serverless -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}' | grep registry.redhat.io
oc get pods -n knative-eventing -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}' | grep registry.redhat.io


# 7. Wait until KnativeEventing and KnativeKafka are ready
oc wait knativeeventings -n knative-eventing knative-eventing --for=condition=Ready=true --timeout=5m
oc wait knativekafkas    -n knative-eventing knative-kafka    --for=condition=Ready=true --timeout=5m
```
