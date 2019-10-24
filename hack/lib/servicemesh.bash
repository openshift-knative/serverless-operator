#!/usr/bin/env bash

function ensure_service_mesh_installed {
  local istio_hostname istio_ip
  logger.info 'Checking if Service Mesh is installed...'

  if oc get namespaces istio-system > /dev/null 2>&1; then
    istio_hostname=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath="{.status.loadBalancer.ingress[0].hostname}")
    istio_ip=$(resolve_hostname "${istio_hostname}")
    if [[ "${istio_ip}" != "" ]]; then
      logger.success 'Service Mesh seems operational. Skipping installation.'
      setup_service_mesh_member_roll
      return 0
    fi
  fi
  install_service_mesh
}

function install_service_mesh {
  logger.info 'Installing ServiceMesh'

  logger.info 'Install the ServiceMesh Operator'
  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: CatalogSourceConfig
metadata:
  name: ci-operators
  namespace: openshift-marketplace
spec:
  targetNamespace: openshift-operators
  packages: elasticsearch-operator,jaeger-product,kiali-ossm,servicemeshoperator
  source: redhat-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch-operator
  namespace: openshift-operators
spec:
  channel: preview
  name: elasticsearch-operator
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger-product
  namespace: openshift-operators
spec:
  channel: stable
  name: jaeger-product
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kiali-ossm
  namespace: openshift-operators
spec:
  channel: stable
  name: kiali-ossm
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: servicemeshoperator
  namespace: openshift-operators
spec:
  channel: "1.0"
  name: servicemeshoperator
  source: ci-operators
  sourceNamespace: openshift-operators
EOF

  logger.info 'Wait for the istio-operator pod to appear'
  timeout 900 "[[ \$(oc get pods -n openshift-operators 2>/dev/null | grep -c istio-operator) -eq 0 ]]" || return 1

  logger.info 'Wait until the Operator pod is up and running'
  wait_until_pods_running openshift-operators || return 1

  logger.info 'Deploy ServiceMesh'
  oc create ns istio-system
  cat <<EOF | oc apply -f -
apiVersion: maistra.io/v1
kind: ServiceMeshControlPlane
metadata:
  name: minimal-multitenant-cni-install
  namespace: istio-system
spec:
  istio:
    global:
      multitenant: true
      proxy:
        autoInject: disabled
      omitSidecarInjectorConfigMap: true
      disablePolicyChecks: false
      defaultPodDisruptionBudget:
        enabled: false
    istio_cni:
      enabled: true
    gateways:
      istio-ingressgateway:
        autoscaleEnabled: false
        type: LoadBalancer
      istio-egressgateway:
        enabled: false
      cluster-local-gateway:
        autoscaleEnabled: false
        enabled: true
        labels:
          app: cluster-local-gateway
          istio: cluster-local-gateway
        ports:
          - name: status-port
            port: 15020
          - name: http2
            port: 80
            targetPort: 8080
          - name: https
            port: 443
    mixer:
      enabled: false
      policy:
        enabled: false
      telemetry:
        enabled: false
    pilot:
      autoscaleEnabled: false
      sidecar: false
    kiali:
      enabled: false
    tracing:
      enabled: false
    prometheus:
      enabled: false
    grafana:
      enabled: false
    sidecarInjectorWebhook:
      enabled: false
EOF
  setup_service_mesh_member_roll

  logger.info 'Wait for the ServiceMeshControlPlane to be ready'
  timeout 900 "[[ \$(oc get smcp -n istio-system -o=jsonpath='{.items[0].status.conditions[?(@.type==\"Ready\")].status}') != 'True' ]]" || return 1

  logger.info 'Wait for Istio Ingressgateway to have external IP'
  wait_until_service_has_external_ip istio-system istio-ingressgateway || fail_test "Ingress has no external IP"
  logger.info 'Wait for Istio Ingressgateway to have DNS resolvable hostname (FQDN)'
  wait_until_hostname_resolves "$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath="{.status.loadBalancer.ingress[0].hostname}")"

  logger.info 'Wait for all pods are running for Istio'
  wait_until_pods_running istio-system

  logger.success "ServiceMesh installed successfully"
}

function setup_service_mesh_member_roll {
  local checkcmd namespaces_s namespaces_joined
  
  namespaces_s="${SERVICE_MESH_MEMBERS[*]}"
  logger.info "Check if Service Mesh Member Roll is configured..."
  if [[ "$(oc get smmr default -n istio-system -o jsonpath='{.status.configuredMembers}')" == "[${namespaces_s}]" ]]; then
    logger.success "Service Mesh Member Roll is configured properly."
    return 0
  fi
  printf -v namespaces_joined "   - %s\n" "${SERVICE_MESH_MEMBERS[@]}"
  namespaces_joined=${namespaces_joined%?}
  logger.info "Adding Service Mesh Member Roll for namespaces: ${namespaces_s}"
  cat <<EOF | oc apply -f - || return $?
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
${namespaces_joined}
EOF
  logger.info "MAISTRA-862 Wait, after namespaces has been added to the members"
  checkcmd="[[ \"\$(oc get smmr default -n istio-system -o jsonpath='{.status.configuredMembers}')\" == '[${namespaces_s}]' ]]"
  timeout 600 "${checkcmd}" || return $?
  logger.success "Service Mesh Member Roll has been configured."
}

function teardown_service_mesh_member_roll {
  logger.info "Teardown of Service Mesh Member Roll (MAISTRA-862)"
  local checkcmd
  if ! oc get smmr default -n istio-system > /dev/null 2>&1; then
    logger.debug 'Service Mesh Member Roll is not present, skipping teardown.'
    return 0
  fi
  if [[ "$(oc get smmr default -n istio-system -o jsonpath='{.spec.members}')" == '[]' ]]; then
    logger.debug 'Service Mesh Member Roll has empty members, skipping teardown.'
    return 0
  fi
  cat <<EOF | oc apply -f - || return $?
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members: []
EOF
  checkcmd="[[ \"\$(oc get smmr default -n istio-system -o jsonpath='{.status.configuredMembers}')\" != '' ]]"
  timeout 600 "${checkcmd}" || return $?
  checkcmd="[[ \"\$(oc get smmr default -n istio-system -o jsonpath='{.spec.members}')\" != '[]' ]]"
  timeout 600 "${checkcmd}" || return $?
  logger.success "Service Mesh Member Roll has been teardowned."
}
