#!/usr/bin/env bash
  
istio_deployments="istio-citadel istio-galley istio-pilot istio-sidecar-injector kiali"
mesh_deployments="elasticsearch-operator istio-operator jaeger-operator kiali-operator"

function install_mesh {
  deploy_servicemesh_operators
  deploy_smcp
  add_smmr
}

function uninstall_mesh {
  remove_smmr

  undeploy_smcp
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  logger.info "Installing service mesh operators in namespace openshift-operators"
  cat <<EOF | oc apply -f - || return $?
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch-operator
  namespace: openshift-operators
spec:
  channel: preview
  name: elasticsearch-operator
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger-product
  namespace: openshift-operators
spec:
  channel: stable
  name: jaeger-product
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kiali-ossm
  namespace: openshift-operators
spec:
  channel: stable
  name: kiali-ossm
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: servicemeshoperator
  namespace: openshift-operators
spec:
  channel: "1.0"
  name: servicemeshoperator
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF

  logger.info "Waiting until service mesh operators are available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators ${mesh_deployments} --no-headers | wc -l) != 4 ]]" || return 1
  oc wait --for=condition=Available deployment ${mesh_deployments} --timeout=300s -n openshift-operators || return $?
}


function deploy_smcp {
  oc create namespace istio-system -o yaml --dry-run | oc apply -f -

  cat <<EOF | oc apply -f - || return $?
apiVersion: maistra.io/v1
kind: ServiceMeshControlPlane
metadata:
  name: basic-install
  namespace: istio-system
spec:

  istio:
    global:
      proxy:
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 128Mi

    gateways:
      istio-egressgateway:
        enabled: false
      istio-ingressgateway:
        enabled: false

    mixer:
      policy:
        autoscaleEnabled: false
      telemetry:
        autoscaleEnabled: false
        resources:
          requests:
            cpu: 100m
            memory: 1G
          limits:
            cpu: 500m
            memory: 4G

    pilot:
      autoscaleEnabled: false
    kiali:
      enabled: true
    grafana:
      enabled: true
    tracing:
      enabled: true
    prometheus:
      enabled: true
EOF

  logger.info "Waiting until service mesh deployments are available"
  timeout 600 "[[ \$(oc get deploy -n istio-system ${istio_deployments} --no-headers | wc -l) != 5 ]]" || return 1
  oc wait --for=condition=Available deployment ${istio_deployments} --timeout=300s -n istio-system || return $?
}


# add_smmr adds ServiceMeshMemberRoll, Networkpolicy and label.
# It needs to call after deploy knative-serving.
function add_smmr {
  cat <<EOF | oc apply -f -
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
    # a list of projects joined into the service mesh
    - default
EOF

  cat <<EOF | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-serving-system-namespace
  namespace: default
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          serving.knative.openshift.io/system-namespace: "true"
  podSelector: {}
  policyTypes:
  - Ingress
EOF

  oc label namespace knative-serving serving.knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace knative-serving-ingress serving.knative.openshift.io/system-namespace=true --overwrite || true
}

function remove_smmr {
  oc delete servicemeshmemberroll default                             -n istio-system  --ignore-not-found
  oc delete networkpolicy         allow-from-serving-system-namespace -n default       --ignore-not-found

  oc label namespace knative-serving serving.knative.openshift.io/system-namespace- --overwrite         || true
  oc label namespace knative-serving-ingress serving.knative.openshift.io/system-namespace- --overwrite || true
}


function undeploy_smcp {
  oc delete servicemeshcontrolplane basic-install -n istio-system --ignore-not-found
  oc wait --for=delete deployment ${istio_deployments} --timeout=300s -n istio-system || true  # Ignore not found error
  timeout 600 "[[ \$(oc get deploy -n istio-system ${istio_deployments} --no-headers | wc -l) != 0 ]]" || return 1
}

function undeploy_servicemesh_operators {
  logger.info "Deleting subscriptions"
  oc delete subscription -n openshift-operators servicemeshoperator kiali-ossm jaeger-product elasticsearch-operator --ignore-not-found
}
