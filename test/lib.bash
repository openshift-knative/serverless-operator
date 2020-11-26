#!/usr/bin/env bash

# == Overrides & test related

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
NAMESPACES+=("${TEST_NAMESPACE}")
NAMESPACES+=("serverless-tests2")
NAMESPACES+=("serverless-tests3")

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/serving.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing-contrib.bash"

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

# Overwritten, safe, version of test function from test-infra that acts well
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
    "$@" || failed=$?

  wait_for_knative_serving_ingress_ns_deleted || return $?

  return $failed
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

  local failed=0

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2ekafka \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=$?

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
    "$@" || failed=$?

  return $failed
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

  local failed=0

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/extensione2e/kafka \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=$?

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
    "$@" || failed=$?

  return $failed
}

# == Upgrade testing

function run_rolling_upgrade_tests {
  logger.info "Running rolling upgrade tests"

  local latest_cluster_version latest_serving_version latest_eventing_version \
    rootdir scope serving_in_scope eventing_in_scope serving_prober_pid \
    eventing_prober_pid prev_serving_version prev_eventing_version retcode

  scope="${1:?Provide an upgrade scope as arg[1]}"
  serving_in_scope="$(echo "${scope}" | grep -vq serving ; echo "$?")"
  eventing_in_scope="$(echo "${scope}" | grep -vq eventing ; echo "$?")"

  prev_serving_version="$(actual_serving_version)"
  prev_eventing_version="$(actual_eventing_version)"

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  if (( eventing_in_scope )); then
    prepare_knative_eventing_tests || return $?
  fi
  if (( serving_in_scope )); then
    prepare_knative_serving_tests || return $?
  fi

  logger.info 'Testing with pre upgrade tests'

  if (( serving_in_scope )); then
    run_serving_preupgrade_test || return $?
  fi
  if (( eventing_in_scope )); then
    run_eventing_preupgrade_test || return $?
  fi

  logger.info 'Starting prober tests'

  if (( serving_in_scope )); then
    start_serving_prober "${prev_serving_version}" /tmp/prober-pid
    retcode=$?
    serving_prober_pid=$(cat /tmp/prober-pid)
    if (( retcode )); then
      return $retcode
    fi
  fi
  if (( eventing_in_scope )); then
    start_eventing_prober /tmp/prober-pid
    retcode=$?
    eventing_prober_pid=$(cat /tmp/prober-pid)
    if (( retcode )); then
      return $retcode
    fi
  fi

  if (( serving_in_scope )); then
    wait_for_serving_prober_ready || return $?
  fi
  if (( eventing_in_scope )); then
    wait_for_eventing_prober_ready || return $?
  fi

  if [[ $UPGRADE_SERVERLESS == true ]]; then
    latest_serving_version="${KNATIVE_SERVING_VERSION/v/}"
    latest_eventing_version="${KNATIVE_EVENTING_VERSION/v/}"

    logger.info "Updating Serverless to ${CURRENT_CSV}"
    logger.debug "Serving version: ${prev_serving_version} -> ${latest_serving_version}"
    logger.debug "Eventing version: ${prev_eventing_version} -> ${latest_eventing_version}"

    approve_csv "$CURRENT_CSV" "$OLM_UPGRADE_CHANNEL"
    if (( serving_in_scope )); then
      check_serving_upgraded "${latest_serving_version}" || return $?
    fi
    if (( eventing_in_scope )); then
      check_eventing_upgraded "${latest_eventing_version}" || return $?
    fi
  fi

  # Might not work in OpenShift CI but we want it here so that we can consume
  # this script later and re-use
  if [[ $UPGRADE_CLUSTER == true ]]; then
    # End the prober test now before we start cluster upgrade, up until now we
    # should have zero failed requests. Cluster upgrade will fail probers as
    # stuff is moved around.
    if (( serving_in_scope )); then
      end_serving_prober "${serving_prober_pid}" || return $?
    fi
    if (( eventing_in_scope )); then
      end_eventing_prober "${eventing_prober_pid}" || return $?
    fi

    upgrade_ocp_cluster "${UPGRADE_OCP_IMAGE:-}" || return $?
  fi

  if (( serving_in_scope )); then
    wait_for_serving_test_services_settle || return $?
  fi

  logger.info "Running postupgrade tests"

  if (( serving_in_scope )); then
    run_serving_postupgrade_test || return $?
  fi
  if (( eventing_in_scope )); then
    run_eventing_postupgrade_test || return $?
  fi

  if (( serving_in_scope )); then
    end_serving_prober "${serving_prober_pid}" || return $?
  fi
  if (( eventing_in_scope )); then
    end_eventing_prober "${eventing_prober_pid}" || return $?
  fi

  cleanup_serving_test_services || return $?

  cd "$rootdir" || return $?
  return 0
}

function end_prober {
  local prober_pid prober_signal retcode title piddir
  title=${1:?Pass a title as arg[1]}
  prober_pid=${2:?Pass a pid as a arg[2]}
  prober_signal=${3:-/tmp/prober-signal}
  piddir="${piddir:-/tmp/svls-probes/$$}"

  mkdir -p "${piddir}" || return $?

  if [ -f "${piddir}/${prober_pid}" ]; then
    logger.info "Prober of PID ${prober_pid} is closed already."
    return 0
  fi
  logger.info "Waiting for ${title} prober test to finish"
  echo 'done' > "${prober_signal}"

  wait "${prober_pid}"
  retcode=$?
  echo 'done' > "${piddir}/${prober_pid}"
  if ! (( retcode )); then
    logger.success "${title} prober passed"
  else
    logger.error "${title} prober failed"
  fi
  return $retcode
}

function upgrade_ocp_cluster {
  local upgrade_ocp_image latest_cluster_version
  upgrade_ocp_image="${1:-}"

  if [[ -n "$upgrade_ocp_image" ]]; then
    oc adm upgrade --to-image="${UPGRADE_OCP_IMAGE}" \
      --force=true --allow-explicit-upgrade || return $?
    timeout 7200 "[[ \$(oc get clusterversion version -o jsonpath='{.status.history[?(@.image==\"${upgrade_ocp_image}\")].state}') != Completed ]]" || return $?
  else
    latest_cluster_version=$(oc adm upgrade | sed -ne '/VERSION/,$ p' \
      | grep -v VERSION | awk '{print $1}' | sort -r | head -n 1)
    [[ $latest_cluster_version != "" ]] || return 1
    oc adm upgrade --to-latest=true --force=true || return $?
    timeout 7200 "[[ \$(oc get clusterversion version -o=jsonpath='{.status.history[?(@.version==\"${latest_cluster_version}\")].state}') != Completed ]]" || return $?
  fi

  logger.success "New cluster version: $(oc get clusterversion \
    version -o jsonpath='{.status.desired.version}')"
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
    htpasswd -b users.htpasswd "user${i}" "password${i}" || return $?
  done

  kubectl create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | kubectl apply -f - || return $?
  oc apply -f openshift/identity/htpasswd.yaml || return $?

  logger.info 'Generate kubeconfig for each user'
  for i in $(seq 1 $num_users); do
    cp "${KUBECONFIG}" "user${i}.kubeconfig"
    occmd="bash -c '! oc login --kubeconfig=user${i}.kubeconfig --username=user${i} --password=password${i} --insecure-skip-tls-verify=true > /dev/null'"
    timeout 180 "${occmd}" || return $?
  done
}

function add_roles {
  logger.info "Adding roles to users"
  oc adm policy add-role-to-user admin user1 -n "$TEST_NAMESPACE" || return $?
  oc adm policy add-role-to-user edit user2 -n "$TEST_NAMESPACE" || return $?
  oc adm policy add-role-to-user view user3 -n "$TEST_NAMESPACE" || return $?
}

function delete_users {
  local user
  logger.info "Deleting users"
  while IFS= read -r line; do
    logger.debug "htpasswd user line: ${line}"
    user=$(echo "${line}" | cut -d: -f1)
    if [ -f "${user}.kubeconfig" ]; then
      rm -fv "${user}.kubeconfig" || return $?
    fi
  done < "users.htpasswd"
  rm -fv users.htpasswd  || return $?
}

function add_systemnamespace_label {
  oc label namespace knative-serving knative.openshift.io/system-namespace=true --overwrite         || true
  oc label namespace knative-serving-ingress knative.openshift.io/system-namespace=true --overwrite || true
}

function add_networkpolicy {
  cat <<EOF | oc apply -f - || return $?
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: deny-by-default
  namespace: "$1"
spec:
  podSelector:
---
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
  echo -n "Waiting for a leader Controller"
  for i in {1..150}; do  # timeout after 5 minutes
    local leader
    leader=$(oc get lease -n "${SERVING_NAMESPACE}" -ojsonpath='{range .items[*].spec}{"\n"}{.holderIdentity}' | cut -d"_" -f1 | grep "^controller-" | head -1)
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
