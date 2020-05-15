#!/usr/bin/env bash

# == Overrides & test releated

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
NAMESPACES+=("${TEST_NAMESPACE}")
NAMESPACES+=("serverless-tests2")

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

  logger.info "Running tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  local failed=0

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/servinge2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=1

  print_test_result ${failed}

  return $failed
}

# Setup a temporary GOPATH to safely check out the repository without breaking other things.
# CAUTION: function overrides GOPATH so use it in subshell or restore original value!
function make_temporary_gopath {
  local tmp_gopath
  tmp_gopath="$(mktemp -d -t gopath-XXXXXXXXXX)"
  ORIGINAL_GOPATH="$(go env GOPATH)"
  export ORIGINAL_GOPATH
  export ORIGINAL_PATH="${PATH}"
  if [[ -d "$(go env GOPATH)/bin" ]]; then
    cp -rv "$(go env GOPATH)/bin" "${tmp_gopath}"
  fi
  logger.info "Temporary GOPATH is: ${tmp_gopath}"
  export GOPATH="$tmp_gopath"
  export PATH="${GOPATH}/bin:${PATH}"
}

function remove_temporary_gopath {
  if [[ "$GOPATH" =~ .*gopath-[0-9a-zA-Z]{10} ]]; then
    logger.info "Removing GOPATH: ${GOPATH}"
    rm -rf "${GOPATH}"
  fi
  if [[ -n "${ORIGINAL_PATH}" ]]; then
    export PATH="${ORIGINAL_PATH}"
    unset ORIGINAL_PATH
  fi
  if [[ -n "${ORIGINAL_GOPATH}" ]]; then
    export GOPATH="${ORIGINAL_GOPATH}"
    unset ORIGINAL_GOPATH
  fi
}

function checkout_repo {
  local target repo version gitref gitdesc targetpath
  target="${1:?Pass target directory as arg[1]}"
  repo="${2:?Pass repository as arg[2]}"
  version="${3:?Pass version as arg[3]}"
  gitref="${4:?Pass git branch as arg[4]}"

  # Setup a temporary GOPATH to safely check out the repository without breaking other things.
  make_temporary_gopath
  # Checkout the relevant code to run
  targetpath="${GOPATH}/src/${target}"
  mkdir -p "$targetpath"
  logger.info "Checking out the ${repo} @ ${version}"
  git clone --branch "${gitref}" \
    --depth 1 \
    "${repo}" \
    "${targetpath}"
  cd "${targetpath}" || return $?
  gitdesc="$(git describe --always --tags --dirty)"
  logger.info "${repo} at ${gitref} has been resolved to ${gitdesc}"
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

  dump_cluster_state
  dump_openshift_olm_state
  dump_openshift_ingress_state
  dump_knative_state
}

function dump_openshift_olm_state {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces || true
  logger.info "Dump of catalog operator log"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator || true
}

function dump_openshift_ingress_state {
  logger.info "Dump of routes.route.openshift.io"
  oc get routes.route.openshift.io -o yaml --all-namespaces || true
  logger.info "Dump of routes.serving.knative.dev"
  oc get routes.serving.knative.dev -o yaml --all-namespaces || true
  logger.info "Dump of openshift-ingress log"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE" || true
}

function dump_knative_state {
  logger.info 'Dump of knative state'
  oc describe knativeserving.operator.knative.dev knative-serving -n "$SERVING_NAMESPACE" || true
  oc describe knativeeventing.operator.knative.dev knative-eventing -n "$EVENTING_NAMESPACE" || true
  oc get pods -n "$SERVING_NAMESPACE" || true
  oc get ksvc --all-namespaces || true
}

# == Test users

function create_htpasswd_users {
  local occmd num_users
  num_users=3
  logger.info "Creating htpasswd for ${num_users} users"

  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.info 'Secret htpass-secret already existsed, updating it.'
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
