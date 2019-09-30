#!/usr/bin/env bash

source $(dirname $0)/../vendor/github.com/knative/test-infra/scripts/e2e-tests.sh

set -x

readonly USER=$KUBE_SSH_USER #satisfy flags.go#initializeFlags()
readonly OPENSHIFT_REGISTRY="${OPENSHIFT_REGISTRY:-"registry.svc.ci.openshift.org"}"
readonly INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-"image-registry.openshift-image-registry.svc:5000"}"
readonly TEST_NAMESPACE=serverless-tests
readonly SERVING_NAMESPACE=knative-serving
readonly OPERATORS_NAMESPACE="openshift-operators"
readonly SERVERLESS_OPERATOR="serverless-operator"
readonly CATALOG_SOURCE_FILENAME="catalogsource-ci.yaml"
env

function scale_up_workers(){
  local cluster_api_ns="openshift-machine-api"

  oc get machineset -n ${cluster_api_ns} --show-labels

  # Get the name of the first machineset that has at least 1 replica
  local machineset=$(oc get machineset -n ${cluster_api_ns} -o custom-columns="name:{.metadata.name},replicas:{.spec.replicas}" | grep -e " [1-9]" | head -n 1 | awk '{print $1}')
  # Bump the number of replicas to 6 (+ 1 + 1 == 8 workers)
  oc patch machineset -n ${cluster_api_ns} ${machineset} -p '{"spec":{"replicas":6}}' --type=merge
  wait_until_machineset_scales_up ${cluster_api_ns} ${machineset} 6
}

# Waits until the machineset in the given namespaces scales up to the
# desired number of replicas
# Parameters: $1 - namespace
#             $2 - machineset name
#             $3 - desired number of replicas
function wait_until_machineset_scales_up() {
  echo -n "Waiting until machineset $2 in namespace $1 scales up to $3 replicas"
  for i in {1..150}; do  # timeout after 15 minutes
    local available=$(oc get machineset -n $1 $2 -o jsonpath="{.status.availableReplicas}")
    if [[ ${available} -eq $3 ]]; then
      echo -e "\nMachineSet $2 in namespace $1 successfully scaled up to $3 replicas"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo - "\n\nError: timeout waiting for machineset $2 in namespace $1 to scale up to $3 replicas"
  return 1
}

# Waits until the given hostname resolves via DNS
# Parameters: $1 - hostname
function wait_until_hostname_resolves() {
  echo -n "Waiting until hostname $1 resolves via DNS"
  for i in {1..150}; do  # timeout after 15 minutes
    local output="$(host -t a $1 | grep 'has address')"
    if [[ -n "${output}" ]]; then
      echo -e "\n${output}"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\nERROR: timeout waiting for hostname $1 to resolve via DNS"
  return 1
}

# Loops until duration (car) is exceeded or command (cdr) returns non-zero
function timeout() {
  SECONDS=0; TIMEOUT=$1; shift
  while eval $*; do
    sleep 5
    [[ $SECONDS -gt $TIMEOUT ]] && echo "ERROR: Timed out" && return 1
  done
  return 0
}

function install_service_mesh(){
  header "Installing ServiceMesh"

  # Install the ServiceMesh Operator
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

  # Wait for the istio-operator pod to appear
  timeout 900 '[[ $(oc get pods -n openshift-operators | grep -c istio-operator) -eq 0 ]]' || return 1

  # Wait until the Operator pod is up and running
  wait_until_pods_running openshift-operators || return 1

  # Deploy ServiceMesh
  oc new-project istio-system
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

  # Wait for the ServiceMeshControlPlane to be ready
  timeout 900 '[[ $(oc get smcp -n istio-system -o=jsonpath="{.items[0].status.conditions[?(@.type==\"Ready\")].status}") != "True" ]]' || return 1

  wait_until_service_has_external_ip istio-system istio-ingressgateway || fail_test "Ingress has no external IP"
  wait_until_hostname_resolves $(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath="{.status.loadBalancer.ingress[0].hostname}")

  wait_until_pods_running istio-system

  header "ServiceMesh installed successfully"
}

function install_catalogsource(){
  header "Installing CatalogSource"

  local serverless_image=$(tag_serverless_operator_image)
  if [[ -n "${serverless_image}" ]]; then
    ./hack/catalog.sh | sed -e "s+\(.* containerImage:\)\(.*\)+\1 ${serverless_image}+g" > $CATALOG_SOURCE_FILENAME
  else
    ./hack/catalog.sh > $CATALOG_SOURCE_FILENAME
  fi
  oc apply -n $OPERATORS_NAMESPACE -f $CATALOG_SOURCE_FILENAME || return 1

  header "CatalogSource installed successfully"
}

function tag_serverless_operator_image(){
  if [[ -n "${OPENSHIFT_BUILD_NAMESPACE}" ]]; then
    oc policy add-role-to-group system:image-puller system:serviceaccounts:${OPERATORS_NAMESPACE} --namespace=${OPENSHIFT_BUILD_NAMESPACE}
    oc tag --insecure=false -n ${OPERATORS_NAMESPACE} ${OPENSHIFT_REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:${SERVERLESS_OPERATOR} ${SERVERLESS_OPERATOR}:latest
    echo $INTERNAL_REGISTRY/$OPERATORS_NAMESPACE/$SERVERLESS_OPERATOR
  fi
}

function create_namespaces(){
  oc new-project $TEST_NAMESPACE
  oc new-project $SERVING_NAMESPACE
}

function run_e2e_tests(){
  header "Running tests"
  go test -v -tags=e2e -count=1 -timeout=10m -parallel=1 ./test/e2e \
      --kubeconfig "$KUBECONFIG" || return 1
}

function delete_catalog_source() {
  echo ">> Deleting CatalogSource"
  oc delete --ignore-not-found=true -n $OPERATORS_NAMESPACE -f $CATALOG_SOURCE_FILENAME
}

function delete_namespaces(){
  echo ">> Deleting namespaces"
  oc delete project $TEST_NAMESPACE
  oc delete project $SERVING_NAMESPACE
}

function teardown() {
  delete_namespaces
  delete_catalog_source
  #TODO: teardown servicemesh ???
}

function dump_openshift_olm_state(){
  echo ">>> subscriptions.operators.coreos.com:"
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces   # This is for status checking.
  echo ">>> catalog operator log:"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator
}

function dump_openshift_ingress_state(){
  echo ">>> routes.route.openshift.io:"
  oc get routes.route.openshift.io -o yaml --all-namespaces
  echo ">>> routes.serving.knative.dev:"
  oc get routes.serving.knative.dev -o yaml --all-namespaces
  echo ">>> openshift-ingress log:"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE"
}

scale_up_workers || exit 1

create_namespaces || exit 1

failed=0

(( !failed )) && install_service_mesh || failed=1

(( !failed )) && install_catalogsource || failed=1

(( !failed )) && run_e2e_tests || failed=1

(( failed )) && dump_cluster_state

(( failed )) && dump_openshift_olm_state

(( failed )) && dump_openshift_ingress_state

teardown

(( failed )) && exit 1

success
