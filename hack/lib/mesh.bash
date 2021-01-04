#!/usr/bin/env bash
  
istio_deployments="istio-citadel istio-galley istio-pilot istio-sidecar-injector kiali"
mesh_deployments=(istio-operator jaeger-operator kiali-operator)

function install_mesh {
  deploy_servicemesh_operators
  deploy_servicemesh_namespace
  deploy_servicemesh_example_certificates
  deploy_smcp
  add_smmr
}

function uninstall_mesh {
  remove_smmr

  undeploy_smcp
  undeploy_servicemesh_example_certificates
  undeploy_servicemesh_namespace
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  logger.info "Installing service mesh operators in namespace openshift-operators"
  cat <<EOF | oc apply -f -
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
  timeout 600 "[[ \$(oc get deploy -n openshift-operators ${mesh_deployments[*]} --no-headers | wc -l) != ${#mesh_deployments[*]} ]]"
  oc wait --for=condition=Available deployment "${mesh_deployments[@]}" --timeout=300s -n openshift-operators
}


function deploy_servicemesh_namespace {
  oc create namespace istio-system -o yaml --dry-run=client | oc apply -f -
}

# This is used to showcase custom domains with TLS, and by TestKsvcWithServiceMeshCustomTlsDomain
function deploy_servicemesh_example_certificates {
  openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 -subj '/O=Example Inc./CN=example.com' -keyout example.com.key -out example.com.crt
  openssl req -out custom.example.com.csr -newkey rsa:2048 -nodes -keyout custom.example.com.key -subj "/CN=custom-ksvc-domain.example.com/O=Example Inc."
  openssl x509 -req -days 365 -CA example.com.crt -CAkey example.com.key -set_serial 0 -in custom.example.com.csr -out custom.example.com.crt

  oc create -n istio-system secret tls custom.example.com \
    --key=custom.example.com.key \
    --cert=custom.example.com.crt \
    -o yaml --dry-run=client | oc apply -f -
  oc create -n istio-system secret tls example.com \
    --key=example.com.key \
    --cert=example.com.crt \
    -o yaml --dry-run=client | oc apply -f -
}


function deploy_smcp {
  cat <<EOF | oc apply -f -
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
        secretVolumes:
        - mountPath: /custom.example.com
          name: custom-example-com
          secretName: custom.example.com

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
  timeout 600 "[[ \$(oc get deploy -n istio-system ${istio_deployments} --no-headers | wc -l) != 5 ]]"
  oc wait --for=condition=Available deployment "${istio_deployments}" --timeout=300s -n istio-system
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
          knative.openshift.io/system-namespace: "true"
  podSelector: {}
  policyTypes:
  - Ingress
EOF

  oc label namespace knative-serving knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace knative-serving-ingress knative.openshift.io/system-namespace=true --overwrite || true
}

function remove_smmr {
  oc delete servicemeshmemberroll default                             -n istio-system  --ignore-not-found
  oc delete networkpolicy         allow-from-serving-system-namespace -n default       --ignore-not-found

  oc label namespace knative-serving knative.openshift.io/system-namespace- --overwrite         || true
  oc label namespace knative-serving-ingress knative.openshift.io/system-namespace- --overwrite || true
}


function undeploy_smcp {
  oc delete servicemeshcontrolplane basic-install -n istio-system --ignore-not-found
  oc wait --for=delete deployment ${istio_deployments} --timeout=300s -n istio-system || true  # Ignore not found error
  timeout 600 "[[ \$(oc get deploy -n istio-system ${istio_deployments} --no-headers | wc -l) != 0 ]]"
}

function undeploy_servicemesh_operators {
  logger.info "Deleting subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator kiali-ossm jaeger-product elasticsearch-operator --ignore-not-found
}

function undeploy_servicemesh_example_certificates {
  oc delete -n istio-system secret example.com --ignore-not-found
  oc delete -n istio-system secret custom.example.com --ignore-not-found
}

function undeploy_servicemesh_namespace {
  oc delete namespace istio-system --ignore-not-found
}
