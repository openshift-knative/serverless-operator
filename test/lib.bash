#!/usr/bin/env bash

# == Overrides & test related

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
declare -a TEST_NAMESPACES
TEST_NAMESPACES=("${TEST_NAMESPACE}" "serverless-tests2" "serverless-tests-mesh")
export TEST_NAMESPACES

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/serving.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing-kafka.bash"

# == Lifefycle

function register_teardown {
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

# Overwritten, safe, version of test function from hack that acts well
# with `set -Eeuo pipefail`.
#
# Run the given E2E tests. Assume tests are tagged e2e, unless `-tags=XXX` is passed.
# Parameters: $1..$n - any go test flags, then directories containing the tests to run.
function go_test_e2e {
  local go_test_args=()
  local retcode
  # Remove empty args as `go test` will consider it as running tests for the
  # current directory, which is not expected.
  [[ ! " $*" == *" -tags="* ]] && go_test_args+=("-tags=e2e")
  for arg in "$@"; do
    [[ -n "$arg" ]] && go_test_args+=("$arg")
  done
  set +Eeuo pipefail
  report_go_test -race -count=1 "${go_test_args[@]}"
  retcode=$?
  set -Eeuo pipefail

  print_test_result "$retcode"
  return "$retcode"
}

function print_test_result {
  local test_status
  test_status="${1:?status is required}"

  if ! (( test_status )); then
    logger.success '🌟 Tests have passed 🌟'
  else
    logger.error '🚨 Tests have failures! 🚨'
  fi
}

function serverless_operator_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running operator e2e tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2e \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}

function serverless_operator_kafka_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Kafka tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2ekafka \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}

function downstream_serving_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Serving tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  # Add system-namespace labels for TestNetworkPolicy and ServiceMesh tests.
  add_systemnamespace_label

  if [[ $FULL_MESH == "true" ]]; then
    export GODEBUG="x509ignoreCN=0"
    go_test_e2e -failfast -timeout=60m -parallel=1 ./test/servinge2e/ \
      --kubeconfig "${kubeconfigs[0]}" \
      --kubeconfigs "${kubeconfigs_str}" \
      "$@"
  else
    go_test_e2e -failfast -timeout=60m -parallel=1 ./test/servinge2e/... \
      --kubeconfig "${kubeconfigs[0]}" \
      --kubeconfigs "${kubeconfigs_str}" \
      "$@"
  fi
}

function downstream_eventing_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Eventing downstream tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/eventinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}

function downstream_knative_kafka_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Knative Kafka tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/extensione2e/kafka \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}

function downstream_monitoring_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Knative monitoring tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/monitoringe2e \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
}

# == Upgrade testing

function run_rolling_upgrade_tests {
  logger.info "Running rolling upgrade tests"

  local image_version image_template channels common_opts

  image_version=$(versions.major_minor "${KNATIVE_SERVING_VERSION}")
  image_template="quay.io/openshift-knative/{{.Name}}:v${image_version}"
  channels=messaging.knative.dev/v1beta1:KafkaChannel,messaging.knative.dev/v1:InMemoryChannel

  # Test configuration. See https://github.com/knative/eventing/tree/main/test/upgrade#probe-test-configuration
  # TODO(ksuszyns): remove EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO when knative/operator#297 is fixed.
  export EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO=false
  export EVENTING_UPGRADE_TESTS_SERVING_USE=true
  export EVENTING_UPGRADE_TESTS_CONFIGMOUNTPOINT=/.config/wathola
  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="${INGRESS_NAMESPACE}"
  export GO_TEST_VERBOSITY=standard-verbose
  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  common_opts=(./test/upgrade "-tags=upgrade" \
    "--kubeconfigs=${KUBECONFIG}" \
    "--channels=${channels}" \
    "--imagetemplate=${image_template}" \
    "--catalogsource=${OLM_SOURCE}" \
    "--upgradechannel=${OLM_UPGRADE_CHANNEL}" \
    "--csv=${CURRENT_CSV}" \
    "--servingversion=${KNATIVE_SERVING_VERSION}" \
    "--eventingversion=${KNATIVE_EVENTING_VERSION}" \
    "--kafkaversion=${KNATIVE_EVENTING_KAFKA_VERSION}" \
    --resolvabledomain \
    --https)

  if [[ "${UPGRADE_SERVERLESS}" == "true" ]]; then
    # TODO: Remove creating the NS when this commit is backported: https://github.com/knative/serving/commit/1cc3a318e185926f5a408a8ec72371ba89167ee7
    oc create namespace serving-tests
    go_test_e2e -run=TestServerlessUpgrade -timeout=30m "${common_opts[@]}"
  fi

  # For reuse in downstream test executions. Might be run after Serverless
  # upgrade or independently.
  if [[ "${UPGRADE_CLUSTER}" == "true" ]]; then
    if oc get namespace serving-tests &>/dev/null; then
      oc delete namespace serving-tests
    fi
    oc create namespace serving-tests
    go_test_e2e -run=TestClusterUpgrade -timeout=190m "${common_opts[@]}" \
      --openshiftimage="${UPGRADE_OCP_IMAGE}" \
      --upgradeopenshift
  fi

  # Delete the leftover namespace with services.
  oc delete namespace serving-tests

  logger.success 'Upgrade tests passed'
}

function teardown {
  if [ -n "$OPENSHIFT_CI" ]; then
    logger.warn 'Skipping teardown as we are running on Openshift CI'
    return 0
  fi
  logger.warn "Teardown 💀"
  teardown_serverless
  teardown_tracing
  # shellcheck disable=SC2153
  delete_namespaces "${SYSTEM_NAMESPACES[@]}" "${TEST_NAMESPACES[@]}"
  delete_catalog_source
  delete_users
}

# == State dumps

function dump_state.setup {
  if (( INTERACTIVE )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi

  error_handlers.register dump_state
}

function dump_state {
  logger.info 'Dumping state...'
  logger.debug 'Environment variables:'
  env

  dump_subscriptions
  gather_knative_state
}

function dump_subscriptions {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces || true
}

function gather_knative_state {
  logger.info 'Gather knative state'
  local gather_dir="${ARTIFACT_DIR:-/tmp}/gather-knative"
  mkdir -p "$gather_dir"
  IMAGE_OPTION=("--image=quay.io/openshift-knative/must-gather")
  if [[ $FULL_MESH == true ]]; then
    IMAGE_OPTION=("${IMAGE_OPTION[@]}" "--image=registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel7")
  fi

  oc --insecure-skip-tls-verify adm must-gather \
    "${IMAGE_OPTION[@]}" \
    --dest-dir "$gather_dir" > "${gather_dir}/gather-knative.log"
}

function check_serverless_alerts {
  logger.info 'Checking Serverless alerts'
  local alerts_file monitoring_route num_alerts
  alerts_file="${ARTIFACTS:-/tmp}/alerts.json"
  monitoring_route=$(oc -n openshift-monitoring get routes alertmanager-main -oyaml -ojsonpath='{.spec.host}')
  # TODO(SRVKE-669) remove the filter for the pingsource-mt-adapter service once issue is fixed.
  curl -k -H "Authorization: Bearer $(oc -n openshift-monitoring sa get-token prometheus-k8s)" \
    "https://${monitoring_route}/api/v1/alerts" | \
    jq -c '.data | map(select((.labels.service != "pingsource-mt-adapter") and (.labels.namespace == "'"${OPERATORS_NAMESPACE}"'" or .labels.namespace == "'"${EVENTING_NAMESPACE}"'" or .labels.namespace == "'"${SERVING_NAMESPACE}"'" or .labels.namespace == "'"${INGRESS_NAMESPACE}"'")))' > "${alerts_file}"

  num_alerts=$(jq 'length' "${alerts_file}")
  if [ ! "${num_alerts}" = "0" ]; then
    echo -e "\n\nERROR: Non-zero number of alerts: ${num_alerts}. Check ${alerts_file}\n"
    jq . "${alerts_file}"
    exit 1
  fi
}

function setup_quick_api_deprecation_alerts {
  local ocp_version
  ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
  # Setup deprecation alerts for OCP >= 4.8
  if versions.le "$(versions.major_minor "$ocp_version")" 4.7; then
    return
  fi
  logger.info "Setup quick API deprecation alerts"
  local namespaces=("${OPERATORS_NAMESPACE}" "${EVENTING_NAMESPACE}" "${SERVING_NAMESPACE}")
  if [[ "${SERVING_NAMESPACE}" != "${INGRESS_NAMESPACE}" ]]; then
    namespaces=("${namespaces[@]}" "${INGRESS_NAMESPACE}")
  fi
  for ns in "${namespaces[@]}"; do
    # Reuse the existing api-usage Prometheus rule and only make it react more quickly.
    oc get prometheusrule api-usage -n openshift-kube-apiserver -oyaml | \
      sed -e "s/\(.*name:.*\)/\1-quick/g" \
          -e "s/\(.*alert:.*\)/\1-quick/g" \
          -e "s/\(.*for:\).*/\1 1m/g" \
          -e "s/\(.*namespace:\).*/\1 ${ns}/g" | oc apply -f -
  done
}

# == Test users

function create_htpasswd_users {
  local occmd num_users
  num_users=${num_users:-3}
  logger.info "Creating htpasswd for ${num_users} users"

  if oc get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.info 'Secret htpass-secret already existed, updating it.'
    # Add a newline to the end of the file if not already present (htpasswd will butcher it otherwise).
    [ -n "$(tail -c1 users.htpasswd)" ] && echo >> users.htpasswd
  else
    touch users.htpasswd
  fi

  logger.info 'Add users to htpasswd'
  for i in $(seq 1 "$num_users"); do
    htpasswd -b users.htpasswd "user${i}" "password${i}"
  done

  oc create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | oc apply -f -

  if oc get oauth.config.openshift.io cluster > /dev/null 2>&1; then
    oc replace -f openshift/identity/htpasswd.yaml
  else
    oc apply -f openshift/identity/htpasswd.yaml
  fi

  logger.info 'Generate kubeconfig for each user'

  if oc config current-context >&/dev/null; then
    ctx=$(oc config current-context)
    cluster=$(oc config view -ojsonpath="{.contexts[?(@.name == \"$ctx\")].context.cluster}")
    server=$(oc config view -ojsonpath="{.clusters[?(@.name == \"$cluster\")].cluster.server}")
    logger.debug "Context: $ctx, Cluster: $cluster, Server: $server"
  else
    # Fallback to in-cluster api server service.
    server="https://kubernetes.default.svc"
  fi

  for i in $(seq 1 "$num_users"); do
    occmd="bash -c '! oc login --insecure-skip-tls-verify=true --kubeconfig=user${i}.kubeconfig --username=user${i} --password=password${i} ${server} > /dev/null'"
    timeout 600 "${occmd}"
  done

  logger.success "${num_users} htpasswd users created"
}

function add_roles {
  logger.info "Adding roles to users"
  oc adm policy add-role-to-user admin user1 -n "$TEST_NAMESPACE"
  oc adm policy add-role-to-user edit user2 -n "$TEST_NAMESPACE"
  oc adm policy add-role-to-user view user3 -n "$TEST_NAMESPACE"
}

function delete_users {
  local user
  logger.info "Deleting users"
  while IFS= read -r line; do
    logger.debug "htpasswd user line: ${line}"
    user=$(echo "${line}" | cut -d: -f1)
    if [ -f "${user}.kubeconfig" ]; then
      rm -fv "${user}.kubeconfig"
    fi
  done < "users.htpasswd"
  rm -fv users.htpasswd
}

function add_systemnamespace_label {
  oc label namespace "$SERVING_NAMESPACE" knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace "$INGRESS_NAMESPACE" knative.openshift.io/system-namespace=true --overwrite || true
}

function add_networkpolicy {
  local NAMESPACE=${1:?Pass a namespace as arg[1]}
  cat <<EOF | oc apply -f -
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: deny-by-default
  namespace: "$NAMESPACE"
spec:
  podSelector:
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-serving-system-namespace
  namespace: "$NAMESPACE"
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
}

function wait_for_leader_controller() {
  local leader
  echo -n "Waiting for a leader Controller"
  for i in {1..150}; do  # timeout after 5 minutes
    local leader
    leader=$(set +o pipefail && oc get lease -n "${SERVING_NAMESPACE}" \
      -ojsonpath='{range .items[*].spec}{"\n"}{.holderIdentity}' \
      | cut -d'_' -f1 | grep "^controller-" | head -1)
    # Make sure the leader pod exists.
    if [ -n "${leader}" ] && oc get pod "${leader}" -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
      echo -e "\nNew leader Controller has been elected"
      return 0
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for leader controller"
  return 1
}

# Sets up a secret in the cert-manager namespace that contains the CA certs that need
# to be trusted to make TLS connections to routes of an arbitrary cluster.
# The Knative test machinery looks for this secret if the --https flag is engaged.
function trust_router_ca() {
  logger.info "Setting up cert-manager/ca-key-pair secret to trust router CA"

  # This is the secret the Knative test machinery looks for if the --https flag is engaged.
  certns="cert-manager"
  certname="ca-key-pair"

  certs=$(mktemp -d)
  oc -n openshift-config-managed get cm default-ingress-cert --template="{{index .data \"ca-bundle.crt\"}}" > "$certs/tls.crt"
  oc get ns $certns || oc create namespace $certns
  oc -n $certns get secret $certname || oc -n $certns create secret generic $certname --from-file=tls.crt="$certs/tls.crt"
}
