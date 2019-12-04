#!/usr/bin/env bash

# == Overrides & test releated

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
readonly TEARDOWN="${TEARDOWN:-on_exit}"
NAMESPACES+=("${TEST_NAMESPACE}")
NAMESPACES+=("serverless-tests2")

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

function run_e2e_tests {
  declare -al kubeconfigs
  local kubeconfigs_str
  
  logger.info "Running tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go test -v -tags=e2e -count=1 -timeout=30m -parallel=1 ./test/e2e \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    && logger.success 'Tests has passed' && return 0 \
    || logger.error 'Tests have failures!' \
    && return 1
}

function run_knative_serving_tests {
  (
  # Setup a temporary GOPATH to safely check out the repository without breaking other things.
  local tmp_gopath
  tmp_gopath="$(mktemp -d -t gopath-XXXXXXXXXX)"
  cp -r "$GOPATH/bin" "$tmp_gopath"
  export GOPATH="$tmp_gopath"

  # save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  # Checkout the relevant code to run
  mkdir -p "$GOPATH/src/knative.dev"
  cd "$GOPATH/src/knative.dev" || return $?
  git clone -b "release-$1" --single-branch https://github.com/openshift/knative-serving.git serving
  cd serving || return $?

  # Remove unneeded manifest
  rm test/config/100-istio-default-domain.yaml

  # Create test resources (namespaces, configMaps, secrets)
  oc apply -f test/config
  oc adm policy add-scc-to-user privileged -z default -n serving-tests
  oc adm policy add-scc-to-user privileged -z default -n serving-tests-alt
  # adding scc for anyuid to test TestShouldRunAsUserContainerDefault.
  oc adm policy add-scc-to-user anyuid -z default -n serving-tests

  local failed=0
  export GATEWAY_NAMESPACE_OVERRIDE="knative-serving-ingress"

  # rolling upgrade tests must run first because they upgrade Serverless to the latest version
  run_knative_serving_rolling_upgrade_tests || failed=1

  run_knative_serving_e2e_and_conformance_tests || failed=1

  rm -rf "$tmp_gopath"
  return $failed
  )
}

function run_knative_serving_e2e_and_conformance_tests {
  local failed=0
  logger.info "Running E2E and conformance tests"
  go test -v -tags=e2e -count=1 -timeout=30m -parallel=3 ./test/e2e ./test/conformance/... \
    --resolvabledomain --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=1
  
  return $failed
}

function run_knative_serving_rolling_upgrade_tests {
  logger.info "Running rolling upgrade tests"
  local failed=0

  go test -tags=preupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || failed=1

  logger.info "Starting prober test"

  rm -f /tmp/prober-signal
  go test -tags=probe -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain &

  # Wait for the upgrade-probe kservice to be ready before proceeding
  timeout 900 '[[ $(oc get ksvc upgrade-probe -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1

  PROBER_PID=$!

  if [[ $UPGRADE_SERVERLESS == true ]]; then
    local serving_version=$(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}")

    # Get the current/latest CSV
    local upgrade_to=$(${rootdir}/hack/catalog.sh | grep currentCSV | awk '{ print $2 }')
    approve_csv $upgrade_to || return 1

    # The knativeserving CR should be updated now
    timeout 900 '[[ ! ( $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") != $serving_version && $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") == True ) ]]' || return 1

    end_prober_test ${PROBER_PID}
  fi

  # Might not work in OpenShift CI but we want it here so that we can consume this script later and re-use
  if [[ $UPGRADE_CLUSTER == true ]]; then
    # End the prober test now before we start cluster upgrade, up until now we should have zero failed requests
    end_prober_test ${PROBER_PID}

    local latest_cluster_version=$(oc adm upgrade | sed -ne '/VERSION/,$ p' | grep -v VERSION | awk '{print $1}')
    [[ $latest_cluster_version == "" ]] && return 1

    oc adm upgrade --to-latest=true

    timeout 7200 '[[ $(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.version==\"${latest_cluster_version}\")].state}") != Completed ]]' || return 1

    logger.info "New cluster version\n: $(oc get clusterversion)"
  fi

  for kservice in `oc get ksvc -n serving-tests --no-headers -o name`; do
    timeout 900 '[[ $(oc get $kservice -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1
  done

  logger.info "Running postupgrade tests"
  go test -tags=postupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || failed=1

  oc delete ksvc pizzaplanet-upgrade-service scale-to-zero-upgrade-service upgrade-probe -n serving-tests

  return $failed
}

function end_prober_test {
  local PROBER_PID=$1
  echo "done" > /tmp/prober-signal
  logger.info "Waiting for prober test to finish"
  wait ${PROBER_PID}
}

function teardown {
  if [ -z "$OPENSHIFT_BUILD_NAMESPACE" ]; then
    logger.warn 'Skipping teardown as we are running on Openshift CI'
    return 0
  fi
  logger.warn "Teardown 💀"
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
}

function dump_openshift_olm_state {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces
  logger.info "Dump of catalog operator log"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator
}

function dump_openshift_ingress_state {
  logger.info "Dump of routes.route.openshift.io"
  oc get routes.route.openshift.io -o yaml --all-namespaces
  logger.info "Dump of routes.serving.knative.dev"
  oc get routes.serving.knative.dev -o yaml --all-namespaces
  logger.info "Dump of openshift-ingress log"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE"
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
