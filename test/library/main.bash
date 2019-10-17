#!/usr/bin/env bash

# shellcheck source=test/library/loader.bash
source "$(dirname ${BASH_SOURCE[0]})/loader.bash"

loader_flag "${BASH_SOURCE[0]}"
loader_addpath "$(dirname "${BASH_SOURCE[0]}")"

include ui/logger.bash

loader_finish

BUILD_NUMBER=${BUILD_NUMBER:-$(uuidgen)}

# shellcheck source=vendor/github.com/knative/test-infra/scripts/e2e-tests.sh
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh"

readonly KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-"image-registry.openshift-image-registry.svc:5000"}"
readonly TEST_NAMESPACE=serverless-tests
readonly SERVING_NAMESPACE=knative-serving
readonly OPERATORS_NAMESPACE="openshift-operators"
readonly KNATIVE_EVENTING_OPERATOR="knative-eventing-operator"
readonly CATALOG_SOURCE_FILENAME="catalogsource-ci.yaml"
readonly CI="${CI:-$(test -z "${GDMSESSION}"; echo $?)}"
readonly SCALE_UP="${SCALE_UP:-true}"
readonly TEARDOWN="${TEARDOWN:-on_exit}"

function initialize {
  if [[ "${TEARDOWN}" == "on_exit" ]]; then
    logger.debug 'Registering trap for teardown as EXIT'
    trap teardown EXIT
    return 0
  fi
  if [[ "${TEARDOWN}" == "at_start" ]]; then
    teardown
    return 0
  fi
  logger.error "TEARDOWN should only have a one of values: \"on_exit\", \"at_start\", but given: ${TEARDOWN}."
  return 2
}

function scale_up_workers(){
  local cluster_api_ns="openshift-machine-api"
  logger.info 'Scaling cluster up'
  if [[ "${SCALE_UP}" != "true" ]]; then
    logger.info 'Skipping scaling up, because SCALE_UP is set to true.'
    return 0
  fi

  logger.debug 'Get the name of the first machineset that has at least 1 replica'
  local machineset
  machineset=$(oc get machineset -n ${cluster_api_ns} -o custom-columns="name:{.metadata.name},replicas:{.spec.replicas}" | grep -e " [1-9]" | head -n 1 | awk '{print $1}')
  logger.debug "Name found: ${machineset}"

  logger.info 'Bump the number of replicas to 6 (+ 1 + 1 == 8 workers)'
  oc patch machineset -n ${cluster_api_ns} "${machineset}" -p '{"spec":{"replicas":6}}' --type=merge
  wait_until_machineset_scales_up ${cluster_api_ns} "${machineset}" 6
}

# Waits until the machineset in the given namespaces scales up to the
# desired number of replicas
# Parameters: $1 - namespace
#             $2 - machineset name
#             $3 - desired number of replicas
function wait_until_machineset_scales_up() {
  logger.info "Waiting until machineset $2 in namespace $1 scales up to $3 replicas"
  local available
  for i in {1..150}; do  # timeout after 15 minutes
    available=$(oc get machineset -n "$1" "$2" -o jsonpath="{.status.availableReplicas}")
    if [[ ${available} -eq $3 ]]; then
      echo ''
      logger.info "Machineset $2 successfully scaled up to $3 replicas"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for machineset $2 in namespace $1 to scale up to $3 replicas"
  return 1
}

# Waits until the given hostname resolves via DNS
# Parameters: $1 - hostname
function wait_until_hostname_resolves() {
  logger.info "Waiting until hostname $1 resolves via DNS"
  for _ in {1..150}; do  # timeout after 15 minutes
    local ip
    ip="$(resolve_hostname "$1")"
    if [[ "$ip" != "" ]]; then
      echo ''
      logger.info "Resolved as ${ip}"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for hostname $1 to resolve via DNS"
  return 1
}

function resolve_hostname {
  local ip
  ip="$(LANG=C host -t a "${1}" | grep 'has address' | head -n 1 | awk '{print $4}')"
  if [ "${ip}" != "" ]; then
    echo "${ip}"
  fi
}

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout {
  local seconds timeout
  seconds=0
  timeout=$1
  shift
  while eval $*; do
    seconds=$(( seconds + 5 ))
    logger.debug "Execution failed: ${*}. Waiting 5 seconds ($seconds/${timeout})..."
    sleep 5
    [[ $seconds -gt $timeout ]] && logger.error "Timed out of ${timeout} exceeded" && return 1
  done
  return 0
}

function install_service_mesh(){
  logger.info "Installing ServiceMesh"

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
  timeout 900 '[[ $(oc get pods -n openshift-operators | grep -c istio-operator) -eq 0 ]]' || return 1

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
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
  namespace: istio-system
spec:
  members:
  - ${SERVING_NAMESPACE}
  - ${TEST_NAMESPACE}
EOF

  logger.info 'Wait for the ServiceMeshControlPlane to be ready'
  timeout 900 '[[ $(oc get smcp -n istio-system -o=jsonpath="{.items[0].status.conditions[?(@.type==\"Ready\")].status}") != "True" ]]' || return 1

  logger.info 'Wait for Istio Ingressgateway to have external IP'
  wait_until_service_has_external_ip istio-system istio-ingressgateway || fail_test "Ingress has no external IP"
  logger.info 'Wait for Istio Ingressgateway to have DNS resolvable hostname (FQDN)'
  wait_until_hostname_resolves $(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath="{.status.loadBalancer.ingress[0].hostname}")

  logger.info 'Wait for all pods are running for Istio'
  wait_until_pods_running istio-system

  logger.success "ServiceMesh installed successfully"
}

function ensure_service_mesh_installed {
  local istio_hostname istio_ip

  if oc get namespaces istio-system 2> /dev/null; then
    istio_hostname=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath="{.status.loadBalancer.ingress[0].hostname}")
    istio_ip=$(resolve_hostname "${istio_hostname}")
    if [[ "${istio_ip}" != "" ]]; then
      logger.info 'Service Mesh seems operational. Skipping installation.'
      return 0
    fi
  fi
  install_service_mesh
}

function install_catalogsource {
  logger.info "Installing CatalogSource"

  local operator_image
  operator_image=$(tag_operator_image)
  if [[ -n "${operator_image}" ]]; then
    ./hack/catalog.sh | sed -e "s+\(.* containerImage:\)\(.*\)+\1 ${operator_image}+g" > $CATALOG_SOURCE_FILENAME
  else
    ./hack/catalog.sh > $CATALOG_SOURCE_FILENAME
  fi
  oc apply -n $OPERATORS_NAMESPACE -f $CATALOG_SOURCE_FILENAME || return 1

  logger.success "CatalogSource installed successfully"
}

function tag_operator_image(){
  if [[ -n "${OPENSHIFT_BUILD_NAMESPACE:-}" ]]; then
    oc policy add-role-to-group system:image-puller system:serviceaccounts:${OPERATORS_NAMESPACE} --namespace="${OPENSHIFT_BUILD_NAMESPACE}" >/dev/null
    oc tag --insecure=false -n ${OPERATORS_NAMESPACE} "${OPENSHIFT_REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:${KNATIVE_EVENTING_OPERATOR} ${KNATIVE_EVENTING_OPERATOR}:latest" >/dev/null
    echo "$INTERNAL_REGISTRY/$OPERATORS_NAMESPACE/$KNATIVE_EVENTING_OPERATOR"
  fi
}

function create_namespaces(){
  logger.info 'Create namespaces'
  oc create ns $TEST_NAMESPACE
  oc create ns $SERVING_NAMESPACE
}

function run_e2e_tests(){
  logger.info "Running tests"
  go test -v -tags=e2e -count=1 -timeout=10m -parallel=1 ./test/e2e \
    --kubeconfig "${KUBECONFIG},$(pwd)/user1.kubeconfig,$(pwd)/user2.kubeconfig" \
    && logger.success 'Tests has passed' && return 0 \
    || logger.error 'Tests have failures!' \
    && return 1
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource"
  oc delete --ignore-not-found=true -n $OPERATORS_NAMESPACE -f $CATALOG_SOURCE_FILENAME
  rm -v $CATALOG_SOURCE_FILENAME
}

function delete_namespaces {
  logger.info "Deleting namespaces"
  oc delete namespace $TEST_NAMESPACE
  oc delete namespace $SERVING_NAMESPACE
}

function delete_users {
  local user
  logger.info "Deleting users"
  while IFS= read -r line; do
    logger.debug "htpasswd user line: ${line}"
    user=$(echo "${line}" | cut -d: -f1)
    rm -v "${user}.kubeconfig"
  done < "users.htpasswd"
  rm -v users.htpasswd
}

function teardown {
  logger.warn "Teardown 💀"
  delete_namespaces
  delete_catalog_source
  delete_users
}

function dump_openshift_olm_state(){
  logger.info "Dump of subscriptions.operators.coreos.com"
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces   # This is for status checking.
  logger.info "Dump of catalog operator log"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator
}

function dump_openshift_ingress_state(){
  logger.info "Dump of routes.route.openshift.io"
  oc get routes.route.openshift.io -o yaml --all-namespaces
  logger.info "Dump of routes.serving.knative.dev"
  oc get routes.serving.knative.dev -o yaml --all-namespaces
  logger.info "Dump of openshift-ingress log"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE"
}

function dump_state {
  if (( CI )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi
  logger.info 'Environment'
  env
  
  dump_cluster_state
  dump_openshift_olm_state
  dump_openshift_ingress_state
}

function create_htpasswd_users {
  local occmd num_users
  num_users=2
  logger.info "Creating htpasswd for ${num_users} users"

  logger.info 'Add users to htpasswd'
  touch users.htpasswd
  for i in $(seq 1 $num_users); do
    htpasswd -b users.htpasswd user${i} password${i}
  done

  oc create secret generic e2e-htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run -o yaml | kubectl apply -f -
  oc apply -f openshift/identity/htpasswd.yaml

  logger.info 'Generate kubeconfig for each user'
  for i in $(seq 1 $num_users); do
    cp "${KUBECONFIG}" "user${i}.kubeconfig"
    occmd="bash -c '! oc login --config=user${i}.kubeconfig --username=user${i} --password=password${i} > /dev/null'"
    timeout 900 "${occmd}" || return 1
  done
}

function add_roles {
  logger.info "Adding roles to users"
  oc adm policy add-role-to-user edit user1 -n $TEST_NAMESPACE
  oc adm policy add-role-to-user view user2 -n $TEST_NAMESPACE
}
