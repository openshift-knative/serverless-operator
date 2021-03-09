#!/usr/bin/env bash

# == Overrides & test related

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
NAMESPACES+=("${TEST_NAMESPACE}")
NAMESPACES+=("serverless-tests2")
NAMESPACES+=("serverless-tests3")
NAMESPACES+=("serverless-tests-mesh")

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

  wait_for_knative_serving_ingress_ns_deleted
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

  go_test_e2e -failfast -timeout=60m -parallel=1 ./test/servinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@"
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

  local image_version image_template patch channels

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  prepare_knative_eventing_tests
  prepare_knative_serving_tests

  cd "$rootdir"

  # Ensure the generated test case names are short enough for Kubernetes.
  # The upgrade framework prefixes the base names and the generated
  # OpenShift Route hostname exceeds the 63-char limit.
  patch="${KNATIVE_SERVING_HOME}/openshift/patches/001-object.patch"
  git apply "${patch}"

  image_version=$(versions.major_minor "${KNATIVE_SERVING_VERSION}")
  image_template="quay.io/openshift-knative/{{.Name}}:v${image_version}"
  channels=messaging.knative.dev/v1beta1:KafkaChannel,messaging.knative.dev/v1:InMemoryChannel

  # Test configuration. See https://github.com/knative/eventing/tree/main/test/upgrade#probe-test-configuration
  # TODO(ksuszyns): remove E2E_UPGRADE_TESTS_SERVING_SCALETOZERO when knative/operator#297 is fixed.
  E2E_UPGRADE_TESTS_SERVING_SCALETOZERO=false \
  E2E_UPGRADE_TESTS_SERVING_USE=true \
  E2E_UPGRADE_TESTS_CONFIGMOUNTPOINT=/.config/wathola \
  E2E_UPGRADE_TESTS_INTERVAL="50ms" \
  GO_TEST_VERBOSITY=standard-verbose \
  SYSTEM_NAMESPACE=knative-serving \
  go_test_e2e -tags=upgrade -timeout=30m \
    ./test/upgrade \
    -channels="${channels}" \
    --kubeconfigs "${KUBECONFIG}" \
    --imagetemplate "${image_template}" \
    --channel="${OLM_UPGRADE_CHANNEL}" \
    --csv="${CURRENT_CSV}" \
    --servingversion="${KNATIVE_SERVING_VERSION}" \
    --eventingversion="${KNATIVE_EVENTING_VERSION}" \
    --openshiftimage="${UPGRADE_OCP_IMAGE}" \
    --resolvabledomain

  git apply -R "${patch}"

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
  delete_namespaces
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

  oc --insecure-skip-tls-verify adm must-gather \
    --image=quay.io/openshift-knative/must-gather \
    --dest-dir "$gather_dir" > "${gather_dir}/gather-knative.log"
}

# == Test users

function create_htpasswd_users {
  local occmd num_users
  num_users=${num_users:-3}
  logger.info "Creating htpasswd for ${num_users} users"

  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
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

  kubectl create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | kubectl apply -f -

  if oc get oauth.config.openshift.io cluster > /dev/null 2>&1; then
    oc replace -f openshift/identity/htpasswd.yaml
  else
    oc apply -f openshift/identity/htpasswd.yaml
  fi

  logger.info 'Generate kubeconfig for each user'
  for i in $(seq 1 "$num_users"); do
    cp "${KUBECONFIG}" "user${i}.kubeconfig"
    occmd="bash -c '! oc login --kubeconfig=user${i}.kubeconfig --username=user${i} --password=password${i} > /dev/null'"
    timeout 180 "${occmd}"
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
  oc label namespace knative-serving knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace knative-serving-ingress knative.openshift.io/system-namespace=true --overwrite || true
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

function trigger_gc_and_print_knative {
  echo ">>> Knative Servings"
  oc get knativeserving.operator.knative.dev --all-namespaces -o yaml

  echo ">>> Knative Services"
  oc get ksvc --all-namespaces

  echo ">>> Triggering GC"
  for pod in $(oc get pod -n openshift-kube-controller-manager -l kube-controller-manager=true -o custom-columns=name:metadata.name --no-headers); do
    echo "killing pod $pod"
    oc rsh -n openshift-kube-controller-manager "$pod" /bin/sh -c "kill 1"
    sleep 30
  done

  echo "Sleeping so GC can run"
  sleep 120

  echo ">>> Knative Servings"
  oc get knativeserving.operator.knative.dev --all-namespaces -o yaml

  echo ">>> Knative Services"
  oc get ksvc --all-namespaces
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
