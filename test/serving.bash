#!/usr/bin/env bash

function wait_for_knative_serving_ingress_ns_deleted {
  local NS="${SERVING_NAMESPACE}-ingress"
  timeout 180 '[[ $(oc get ns $NS --no-headers | wc -l) == 1 ]]' || true
  # Workaround for https://bugzilla.redhat.com/show_bug.cgi?id=1798282 on Azure - if loadbalancer status is empty
  # it's safe to remove the finalizer.
  if oc -n $NS get svc kourier >/dev/null 2>&1 && [ "$(oc -n $NS get svc kourier -ojsonpath="{.status.loadBalancer.*}")" = "" ]; then
    oc -n $NS patch services/kourier --type=json --patch='[{"op":"replace","path":"/metadata/finalizers","value":[]}]'
  fi
  timeout 180 '[[ $(oc get ns $NS --no-headers | wc -l) == 1 ]]' || return 1
}

function prepare_knative_serving_tests {
  logger.debug 'Preparing Serving tests'

  cd "$KNATIVE_SERVING_HOME" || return $?

  # Create test resources (namespaces, configMaps, secrets)
  oc apply -f test/config
  oc adm policy add-scc-to-user privileged -z default -n serving-tests
  oc adm policy add-scc-to-user privileged -z default -n serving-tests-alt
  # Adding scc for anyuid to test TestShouldRunAsUserContainerDefault.
  oc adm policy add-scc-to-user anyuid -z default -n serving-tests
  # Add networkpolicy to test namespace and label to serving namespaces for testing under the strict networkpolicy.
  add_networkpolicy "serving-tests"
  add_networkpolicy "serving-tests-alt"
  add_systemnamespace_label

  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="${SERVING_NAMESPACE}-ingress"
}

function upstream_knative_serving_e2e_and_conformance_tests {
  logger.info "Running Serving E2E and conformance tests"
  (
  prepare_knative_serving_tests || return $?

  local failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"

  local parallel=3

  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') = VSphere ]]; then
    # Since we don't have LoadBalancers working, gRPC tests will always fail.
    rm ./test/e2e/grpc_test.go
    parallel=2
  fi

  go_test_e2e -tags=e2e -timeout=30m -parallel=$parallel ./test/e2e ./test/conformance/api/... ./test/conformance/runtime/... \
    --resolvabledomain --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=1

  # Run the helloworld test with an image pulled into the internal registry.
  oc tag -n serving-tests "registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-helloworld" "helloworld:latest" --reference-policy=local
  go_test_e2e -tags=e2e -timeout=30m ./test/e2e -run "^(TestHelloWorld)$" \
    --resolvabledomain --kubeconfig "$KUBECONFIG" \
    --imagetemplate "image-registry.openshift-image-registry.svc:5000/serving-tests/{{.Name}}" || failed=2
  
  # Prevent HPA from scaling to make HA tests more stable
  local max_replicas
  max_replicas=$(oc get hpa activator -n "$SERVING_NAMESPACE" -ojsonpath='{.spec.maxReplicas}')
  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch '{"spec":{"maxReplicas":2}}' || failed=3

  # Use sed as the -spoofinterval parameter is not available yet
  sed "s/\(.*requestInterval =\).*/\1 10 * time.Millisecond/" -i test/vendor/knative.dev/pkg/test/spoof/spoof.go

  go_test_e2e -tags=e2e -timeout=15m -failfast -parallel=1 ./test/ha \
    --resolvabledomain \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=4

  # Restore the original maxReplicas for any tests running after this test suite
  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch '{"spec":{"maxReplicas":'${max_replicas}'}}' || failed=5

  print_test_result ${failed}

  return $failed
  )
}

function actual_serving_version {
  oc get knativeserving.operator.knative.dev \
    knative-serving -n "${SERVING_NAMESPACE}" -o=jsonpath="{.status.version}"
}

function run_serving_preupgrade_test {
  logger.info 'Running Serving pre upgrade tests'

  local image_template

  cd "${KNATIVE_SERVING_HOME}" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"

  go_test_e2e -tags=preupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain

  logger.success 'Serving pre upgrade tests passed'
}

function start_serving_prober {
  local image_template prev_serving_version probe_fraction serving_prober_pid \
    result_file
  prev_serving_version="${1:?Pass a previous Serving version as arg[1]}"
  result_file="${2:?Pass a result file as arg[2]}"

  logger.info 'Starting Serving prober'

  rm -fv /tmp/prober-signal
  cd "${KNATIVE_SERVING_HOME}" || return $?

  probe_fraction=1.0
  if [[ ${prev_serving_version} < "0.14.0" ]]; then
    probe_fraction=0.95
  fi
  logger.info "Target success fraction for Serving is ${probe_fraction}"

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"

  go_test_e2e -tags=probe \
    -timeout=30m \
    ./test/upgrade \
    -probe.success_fraction=${probe_fraction} \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain &
  serving_prober_pid=$!

  logger.debug "Serving prober PID is ${serving_prober_pid}"

  echo ${serving_prober_pid} > "${result_file}"
}

function wait_for_serving_prober_ready {
  # Wait for the upgrade-probe kservice to be ready before proceeding
  timeout 900 "[[ \$(oc get services.serving.knative.dev upgrade-probe \
    -n serving-tests -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') \
    != True ]]"

  logger.success 'Serving prober is ready'
}

function check_serving_upgraded {
  local latest_serving_version
  latest_serving_version="${1:?Pass a target serving version as arg[1]}"

  logger.debug 'Check KnativeServing has the latest version with Ready status'
  timeout 300 "[[ ! ( \$(oc get knativeserving.operator.knative.dev \
    knative-serving -n ${SERVING_NAMESPACE} -o=jsonpath='{.status.version}') \
    == ${latest_serving_version} && \$(oc get knativeserving.operator.knative.dev \
    knative-serving -n ${SERVING_NAMESPACE} \
    -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') == True ) ]]"
}

function end_serving_prober {
  local prober_pid
  prober_pid="${1:?Pass a prober pid as arg[1]}"

  end_prober_test 'Serving' "${prober_pid}"
}

function wait_for_serving_test_services_settle {
  # Wait for all services to become ready again. Exclude the upgrade-probe as
  # that'll be removed by the prober test above.
  for kservice in $(oc get ksvc -n serving-tests --no-headers -o name | grep -v 'upgrade-probe'); do
    timeout 900 "[[ \$(oc get ${kservice} -n serving-tests -o jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != True ]]"
  done

  # Give time to settle things down
  sleep 30
}

function run_serving_postupgrade_test {
  logger.info 'Running Serving post upgrade tests'

  local image_template

  cd "${KNATIVE_SERVING_HOME}" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"

  go_test_e2e -tags=postupgrade \
    -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return $?

  logger.success 'Serving post upgrade tests passed'
}

function cleanup_serving_test_servinces {
  oc delete --ignore-not-found=true ksvc \
    pizzaplanet-upgrade-service \
    scale-to-zero-upgrade-service \
    upgrade-probe -n serving-tests
}
