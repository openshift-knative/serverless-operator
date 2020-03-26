#!/usr/bin/env bash

# == Overrides & test releated

# shellcheck disable=SC1091,SC1090
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
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
    --channel "$CHANNEL" \
    --kubeconfig "${kubeconfigs[0]}" \
    --kubeconfigs "${kubeconfigs_str}" \
    "$@" || failed=1

  if (( !failed )); then
    logger.success 'Tests have passed'
  else
    logger.error 'Tests have failures!'
  fi

  wait_for_knative_serving_ingress_ns_deleted || return 1

  return $failed
}

function wait_for_knative_serving_ingress_ns_deleted {
  timeout 180 '[[ $(oc get ns knative-serving-ingress --no-headers | wc -l) == 1 ]]' || true
  # Workaround for https://bugzilla.redhat.com/show_bug.cgi?id=1798282 on Azure - if loadbalancer status is empty
  # it's safe to remove the finalizer.
  if oc -n knative-serving-ingress get svc kourier >/dev/null 2>&1 && [ "$(oc -n knative-serving-ingress get svc kourier -ojsonpath="{.status.loadBalancer.*}")" = "" ]; then
    oc -n knative-serving-ingress patch services/kourier --type=json --patch='[{"op":"replace","path":"/metadata/finalizers","value":[]}]'
  fi
  timeout 180 '[[ $(oc get ns knative-serving-ingress --no-headers | wc -l) == 1 ]]' || return 1
}

# Setup a temporary GOPATH to safely check out the repository without breaking other things.
# CAUTION: function overrides GOPATH so use it in subshell or restore original value!
function make_temporary_gopath {
  local tmp_gopath
  tmp_gopath="$(mktemp -d -t gopath-XXXXXXXXXX)"
  if [[ -d $(go env GOPATH)/bin ]]; then
    cp -rv "$(go env GOPATH)/bin" "${tmp_gopath}"
  fi
  logger.info "Temporary GOPATH is: ${tmp_gopath}"
  export GOPATH="$tmp_gopath"
  export PATH="$GOPATH/bin":$PATH
}

function remove_temporary_gopath {
  if [[ "$GOPATH" =~ .*gopath-[0-9a-zA-Z]{10} ]]; then
    logger.info "Removing GOPATH: ${GOPATH}"
    rm -rf "${GOPATH}"
  fi
}

function checkout_knative_serving {
  local knative_version=$1
  # Setup a temporary GOPATH to safely check out the repository without breaking other things.
  make_temporary_gopath

  # Checkout the relevant code to run
  export KNATIVE_SERVING_HOME="$GOPATH/src/knative.dev/serving"
  mkdir -p "$KNATIVE_SERVING_HOME"
  git clone -b "release-${knative_version}" --depth 1 https://github.com/openshift/knative-serving.git "$KNATIVE_SERVING_HOME"
  git describe --always --tags
}

function prepare_knative_serving_tests {
  # Remove unneeded manifest
  rm test/config/100-istio-default-domain.yaml

  # Create test resources (namespaces, configMaps, secrets)
  oc apply -f test/config
  oc adm policy add-scc-to-user privileged -z default -n serving-tests
  oc adm policy add-scc-to-user privileged -z default -n serving-tests-alt
  # Adding scc for anyuid to test TestShouldRunAsUserContainerDefault.
  oc adm policy add-scc-to-user anyuid -z default -n serving-tests

  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="knative-serving-ingress"
}

function run_knative_serving_e2e_and_conformance_tests {
  logger.info "Running Serving E2E and conformance tests"
  (
  local knative_version=$1

  if [[ -z ${KNATIVE_SERVING_HOME+x} ]]; then
    checkout_knative_serving "$knative_version"
  fi
  cd "$KNATIVE_SERVING_HOME" || return $?

  prepare_knative_serving_tests || return $?

  local failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${knative_version}:knative-serving-test-{{.Name}}"

  local parallel=3

  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') = VSphere ]]; then
    # Since we don't have LoadBalancers working, gRPC tests will always fail.
    rm ./test/e2e/grpc_test.go
    parallel=2
  fi

  go_test_e2e -tags=e2e -timeout=30m -parallel=$parallel ./test/e2e ./test/conformance/api/... ./test/conformance/runtime/... \
    --resolvabledomain --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=1

  remove_temporary_gopath

  return $failed
  )
}

function run_knative_serving_rolling_upgrade_tests {
  logger.info "Running Serving rolling upgrade tests"
  (
  local knative_version=$1

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  if [[ -z ${KNATIVE_SERVING_HOME+x} ]]; then
    checkout_knative_serving "$knative_version"
  fi
  cd "$KNATIVE_SERVING_HOME" || return $?

  prepare_knative_serving_tests || return $?

  local failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${knative_version}:knative-serving-test-{{.Name}}"

  go_test_e2e -tags=preupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return 1

  logger.info "Starting prober test"

  rm -f /tmp/prober-signal
  go_test_e2e -tags=probe -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain &

  # Wait for the upgrade-probe kservice to be ready before proceeding
  timeout 900 '[[ $(oc get services.serving.knative.dev upgrade-probe -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1

  PROBER_PID=$!

  if [[ $UPGRADE_SERVERLESS == true ]]; then
    # Get latest CSV from the given channel
    local upgrade_to
    upgrade_to=$("${rootdir}/hack/catalog.sh" | sed -n '/channels/,$p;' | sed -n "/- name: ${CHANNEL}$/{n;p;}" | awk '{ print $2 }')

    local cluster_version
    cluster_version=$(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.state==\"Completed\")].version}")
    if [[ "$cluster_version" = 4.1.* || "${HOSTNAME}" = *ocp-41* || \
          "$cluster_version" = 4.2.* || "${HOSTNAME}" = *ocp-42* ]]; then
      if approve_csv "$upgrade_to" ; then # Upgrade should fail on OCP 4.1, 4.2
        return 1
      fi
      # Check we got RequirementsNotMet error
      [[ $(oc get ClusterServiceVersion $upgrade_to -n $OPERATORS_NAMESPACE -o=jsonpath="{.status.requirementStatus[?(@.name==\"$upgrade_to\")].message}") =~ "requirement not met: minKubeVersion" ]] || return 1
    else
      approve_csv "$upgrade_to" || return 1
      timeout 900 '[[ ! ( $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") == True ) ]]' || return 1
    fi
    end_prober_test ${PROBER_PID}
  fi

  # Might not work in OpenShift CI but we want it here so that we can consume this script later and re-use
  if [[ $UPGRADE_CLUSTER == true ]]; then
    # End the prober test now before we start cluster upgrade, up until now we should have zero failed requests
    end_prober_test ${PROBER_PID}

    local latest_cluster_version=$(oc adm upgrade | sed -ne '/VERSION/,$ p' | grep -v VERSION | awk '{print $1}' | sort -r | head -n 1)
    [[ $latest_cluster_version != "" ]] || return 1

    oc adm upgrade --to-latest=true

    timeout 7200 '[[ $(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.version==\"${latest_cluster_version}\")].state}") != Completed ]]' || return 1

    logger.info "New cluster version\n: $(oc get clusterversion)"
  fi

  # Wait for all services to become ready again. Exclude the upgrade-probe as that'll be removed by the prober test above.
  for kservice in $(oc get ksvc -n serving-tests --no-headers -o name | grep -v "upgrade-probe"); do
    timeout 900 '[[ $(oc get $kservice -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1
  done

  # Give time to settle things down
  sleep 30

  logger.info "Running postupgrade tests"
  go_test_e2e -tags=postupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return 1

  oc delete ksvc pizzaplanet-upgrade-service scale-to-zero-upgrade-service upgrade-probe -n serving-tests

  remove_temporary_gopath

  return 0
  )
}

function end_prober_test {
  local PROBER_PID=$1
  echo "done" > /tmp/prober-signal
  logger.info "Waiting for prober test to finish"
  wait "${PROBER_PID}"
}

function run_knative_serving_operator_tests {
  (
  local version target serverless_rootdir exitstatus patchfile fork gitdesc
  version=$1
  fork="${2:-openshift-knative}"
  serverless_rootdir="$(pwd)"
  make_temporary_gopath

  logger.info "Checkout the code ${fork}/serving-operator @ ${version}"
  target="${GOPATH}/src/knative.dev/serving-operator"
  mkdir -p "$target"
  git clone --branch "openshift-${version}" --depth 1 \
    "https://github.com/${fork}/serving-operator.git" \
    "${target}"
  pushd "${target}" || return $?

  gitdesc=$(git describe --always --tags --dirty)

  exitstatus=0

  logger.info "Run tests of knative/serving-operator @ ${gitdesc}"

  export TEST_NAMESPACE="knative-serving"
  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2e \
    --kubeconfig "$KUBECONFIG" \
    || exitstatus=5$? && true

  if (( !exitstatus )); then
    logger.success 'Tests have passed'
  else
    logger.error 'Tests have failures!'
  fi

  wait_for_knative_serving_ingress_ns_deleted || return 1

  remove_temporary_gopath

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
  oc describe knativeserving.operator.knative.dev knative-serving -n "$SERVING_NAMESPACE" || true
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
