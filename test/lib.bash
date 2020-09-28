#!/usr/bin/env bash

# == Overrides & test releated

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
NAMESPACES+=("${TEST_NAMESPACE}")
NAMESPACES+=("serverless-tests2")
NAMESPACES+=("serverless-tests3")

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/serving.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing.bash"

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

  logger.info "Running tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  local failed=0

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2e \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=1

  print_test_result ${failed}

  wait_for_knative_serving_ingress_ns_deleted || return 1

  return $failed
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

  local failed=0

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/servinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=1

  print_test_result ${failed}

  return $failed
}

function downstream_eventing_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Eventing tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  local failed=0

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/eventinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=1

  print_test_result ${failed}

  return $failed
}

function end_prober_test {
  local PROBER_PID=$1
  echo "done" > /tmp/prober-signal
  logger.info "Waiting for prober test to finish"
  wait "${PROBER_PID}"
  return $?
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

function dump_state {
  if (( INTERACTIVE )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi
  logger.info 'Environment'
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
  num_users=3
  logger.info "Creating htpasswd for ${num_users} users"

  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.info 'Secret htpass-secret already existed, updating it.'
    # Add a newline to the end of the file if not already present (htpasswd will butcher it otherwise).
    sed -i -e '$a\' users.htpasswd
  else
    touch users.htpasswd
  fi

  logger.info 'Add users to htpasswd'
  for i in $(seq 1 $num_users); do
    htpasswd -b users.htpasswd "user${i}" "password${i}"
  done

  kubectl create secret generic htpass-secret \
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
      rm -v "${user}.kubeconfig"
    fi
  done < "users.htpasswd"
  rm -v users.htpasswd
}

function add_systemnamespace_label {
  oc label namespace knative-serving serving.knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace knative-serving-ingress serving.knative.openshift.io/system-namespace=true --overwrite || true
}

function add_networkpolicy {
  local NAMESPACE=$1
  cat <<EOF | oc apply -f -
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: deny-by-default
  namespace: "$1"
spec:
  podSelector:
EOF

  cat <<EOF | oc apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-serving-system-namespace
  namespace: "$1"
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
  echo -n "Waiting for a leader Controller"
  for i in {1..150}; do  # timeout after 5 minutes
    local leader=$(oc get lease -n "${SERVING_NAMESPACE}" -ojsonpath='{range .items[*].spec}{"\n"}{.holderIdentity}' | cut -d"_" -f1 | grep "^controller-" | head -1)
    # Make sure the leader pod exists.
    if [ -n "${leader}" ] && oc get pod "${leader}" -n "${SERVING_NAMESPACE}"  >/dev/null 2>&1; then
      echo -e "\nNew leader Controller has been elected"
      return 0
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for leader controller"
  return 1
}
