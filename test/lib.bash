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

# Setup a temporary GOPATH to safely check out the repository without breaking other things.
# CAUTION: function overrides GOPATH so use it in subshell or restore original value!
function make_temporary_gopath {
  local tmp_gopath
  tmp_gopath="$(mktemp -d -t gopath-XXXXXXXXXX)"
  cp -rv "$(go env GOPATH)/bin" "${tmp_gopath}"
  logger.info "Temporary GOPATH is: ${tmp_gopath}"
  export GOPATH="$tmp_gopath"
}

function run_knative_serving_tests {
  (
  local knative_version=$1
  make_temporary_gopath

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  # Checkout the relevant code to run
  mkdir -p "$GOPATH/src/knative.dev"
  git clone --branch "release-${knative_version}" \
    --depth=1 https://github.com/openshift/knative-serving.git \
    "${GOPATH}/src/knative.dev/serving"
  pushd "${GOPATH}/src/knative.dev/serving" || return $?

  # Remove unneeded manifest
  rm test/config/100-istio-default-domain.yaml

  # Create test resources (namespaces, configMaps, secrets)
  oc apply -f test/config
  oc adm policy add-scc-to-user privileged -z default -n serving-tests
  oc adm policy add-scc-to-user privileged -z default -n serving-tests-alt
  # Adding scc for anyuid to test TestShouldRunAsUserContainerDefault.
  oc adm policy add-scc-to-user anyuid -z default -n serving-tests

  local failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${knative_version}:knative-serving-test-{{.Name}}"
  export GATEWAY_NAMESPACE_OVERRIDE="knative-serving-ingress"

  git describe --always --tags --dirty

  # Rolling upgrade tests must run first because they upgrade Serverless to the latest version
  if [[ $RUN_KNATIVE_SERVING_UPGRADE_TESTS == true ]]; then
    run_knative_serving_rolling_upgrade_tests || failed=1
  fi

  if [[ $RUN_KNATIVE_SERVING_E2E == true ]]; then
    run_knative_serving_e2e_and_conformance_tests || failed=1
  fi

  rm -rf "${GOPATH}"
  popd || return $?
  return $failed
  )
}

function run_knative_serving_e2e_and_conformance_tests {
  logger.info "Running E2E and conformance tests"
  go test -v -tags=e2e -count=1 -timeout=30m -parallel=3 ./test/e2e ./test/conformance/... \
    --resolvabledomain --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || return 1
}

function run_knative_serving_rolling_upgrade_tests {
  logger.info "Running rolling upgrade tests"

  go test -v -tags=preupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return 1

  logger.info "Starting prober test"

  rm -f /tmp/prober-signal
  go test -v -tags=probe -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain &

  # Wait for the upgrade-probe kservice to be ready before proceeding
  timeout 900 '[[ $(oc get services.serving.knative.dev upgrade-probe -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1

  PROBER_PID=$!

  if [[ $UPGRADE_SERVERLESS == true ]]; then
    local serving_version=$(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}")

    # Get the current/latest CSV
    local upgrade_to=$(${rootdir}/hack/catalog.sh | grep currentCSV | awk '{ print $2 }')

    if [[ ${HOSTNAME} = *ocp-41* ]]; then
      if approve_csv "$upgrade_to" ; then # Upgrade should fail on OCP 4.1
        return 1
      fi
      # Check we got RequirementsNotMet error
      [[ $(oc get ClusterServiceVersion $upgrade_to -n $OPERATORS_NAMESPACE -o=jsonpath="{.status.requirementStatus[?(@.name==\"$upgrade_to\")].message}") =~ "requirement not met: minKubeVersion" ]] || return 1
      # Check KnativeServing still has the old version
      [[ $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") == "$serving_version" ]] || return 1
    else
      approve_csv "$upgrade_to" || return 1
      # The knativeserving CR should be updated now
      timeout 900 '[[ ! ( $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") != $serving_version && $(oc get knativeserving knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") == True ) ]]' || return 1
    fi
    end_prober_test ${PROBER_PID}
  fi

  # Might not work in OpenShift CI but we want it here so that we can consume this script later and re-use
  if [[ $UPGRADE_CLUSTER == true ]]; then
    # End the prober test now before we start cluster upgrade, up until now we should have zero failed requests
    end_prober_test ${PROBER_PID}

    local latest_cluster_version=$(oc adm upgrade | sed -ne '/VERSION/,$ p' | grep -v VERSION | awk '{print $1}')
    [[ $latest_cluster_version != "" ]] || return 1

    oc adm upgrade --to-latest=true

    timeout 7200 '[[ $(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.version==\"${latest_cluster_version}\")].state}") != Completed ]]' || return 1

    logger.info "New cluster version\n: $(oc get clusterversion)"
  fi

  for kservice in `oc get ksvc -n serving-tests --no-headers -o name`; do
    timeout 900 '[[ $(oc get $kservice -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1
  done

  logger.info "Running postupgrade tests"
  go test -v -tags=postupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return 1

  oc delete ksvc pizzaplanet-upgrade-service scale-to-zero-upgrade-service upgrade-probe -n serving-tests

  return 0
}

function end_prober_test {
  local PROBER_PID=$1
  echo "done" > /tmp/prober-signal
  logger.info "Waiting for prober test to finish"
  wait ${PROBER_PID}
}

function run_knative_serving_operator_tests {
  (
  local version target serverless_rootdir exitstatus patchfile fork gitdesc
  version="$1"
  fork="${2:-knative}"
  serverless_rootdir="$(pwd)"
  make_temporary_gopath

  logger.info "Checkout the code ${fork}/serving-operator @ ${version}"
  mkdir -p "${GOPATH}/src/knative.dev"
  target="${GOPATH}/src/knative.dev/serving-operator"
  git clone --branch "${version}" --depth 1 \
    "https://github.com/${fork}/serving-operator.git" \
    "${target}"
  pushd "${target}" || return $?

  patchfile="${serverless_rootdir}/test/patches/SRVKS-241-knative-serving-operator-skip-configure.patch"
  logger.info "Apply walkaround for SRVKS-241"
  logger.debug "Patchfile is: ${patchfile}"
  [ -f "${patchfile}" ] || return $?
  patch --strip=0 < "${patchfile}" || return $?

  gitdesc=$(git describe --always --tags --dirty)

  exitstatus=0

  logger.info "Run tests of knative/serving-operator @ ${gitdesc}"
  env TEST_NAMESPACE='knative-serving' \
    go test -v -tags=e2e -count=1 -timeout=30m -parallel=1 ./test/e2e \
      --kubeconfig "$KUBECONFIG" \
    || exitstatus=5$? && true

  if (( !exitstatus )); then
    logger.success 'Tests have passed'
  else
    logger.error 'Tests have failures!'
  fi

  logger.info "Removing GOPATH: ${GOPATH}"
  rm -rf "${GOPATH}"
  popd || return $?
  return $exitstatus
  )
}

function teardown {
  if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then
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
  oc describe knativeserving knative-serving -n "$SERVING_NAMESPACE" || true
  oc get smcp --all-namespaces || true
  oc get smmr --all-namespaces || true
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
