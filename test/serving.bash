#!/usr/bin/env bash

function wait_for_knative_serving_ingress_ns_deleted {
  timeout 180 '[[ $(oc get ns knative-serving-ingress --no-headers | wc -l) == 1 ]]' || true
  # Workaround for https://bugzilla.redhat.com/show_bug.cgi?id=1798282 on Azure - if loadbalancer status is empty
  # it's safe to remove the finalizer.
  if oc -n knative-serving-ingress get svc kourier >/dev/null 2>&1 && [ "$(oc -n knative-serving-ingress get svc kourier -ojsonpath="{.status.loadBalancer.*}")" = "" ]; then
    oc -n knative-serving-ingress patch services/kourier --type=json --patch='[{"op":"replace","path":"/metadata/finalizers","value":[]}]'
  fi
  timeout 180 '[[ $(oc get ns knative-serving-ingress --no-headers | wc -l) == 1 ]]' || return 1
}

function checkout_knative_serving {
  checkout_repo 'knative.dev/serving' \
    "${KNATIVE_SERVING_REPO}" \
    "${KNATIVE_SERVING_VERSION}" \
    "${KNATIVE_SERVING_BRANCH}"
  export KNATIVE_SERVING_HOME="${GOPATH}/src/knative.dev/serving"
}

function checkout_knative_serving_operator {
  checkout_repo 'knative.dev/serving-operator' \
    "${KNATIVE_SERVING_OPERATOR_REPO}" \
    "${KNATIVE_SERVING_OPERATOR_VERSION}" \
    "${KNATIVE_SERVING_OPERATOR_BRANCH}"
}

function prepare_knative_serving_tests {
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

  if [[ -z ${KNATIVE_SERVING_HOME+x} ]]; then
    checkout_knative_serving
  fi
  cd "$KNATIVE_SERVING_HOME" || return $?

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
  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch '{"spec":{"maxReplicas":2}}' || failed=3

  # Use sed as the -spoofinterval parameter is not available yet
  sed "s/\(.*requestInterval =\).*/\1 10 * time.Millisecond/" -i test/vendor/knative.dev/pkg/test/spoof/spoof.go

  go_test_e2e -tags=e2e -timeout=15m -failfast -parallel=1 ./test/ha \
    --resolvabledomain \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=4

  print_test_result ${failed}

  remove_temporary_gopath

  return $failed
  )
}

function run_knative_serving_rolling_upgrade_tests {
  logger.info "Running Serving rolling upgrade tests"
  (
  local failed upgrade_to latest_cluster_version cluster_version latest_serving_version

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  if [[ -z ${KNATIVE_SERVING_HOME+x} ]]; then
    checkout_knative_serving
  fi
  cd "$KNATIVE_SERVING_HOME" || return $?

  prepare_knative_serving_tests || return $?

  failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"

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
    logger.info "updating serving version from ${PREVIOUS_CSV} to ${CURRENT_CSV}"

    latest_serving_version="${KNATIVE_SERVING_VERSION/v/}"

    # Get latest CSV from the given channel
    upgrade_to="$CURRENT_CSV"

    cluster_version=$(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.state==\"Completed\")].version}")

    approve_csv "$upgrade_to" "$OLM_UPGRADE_CHANNEL" || return 1
    # Check KnativeServing has the latest version with Ready status
    timeout 300 '[[ ! ( $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") == $latest_serving_version && $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") == True ) ]]' || return 1

    end_prober_test ${PROBER_PID} || return $?
  fi

  # Might not work in OpenShift CI but we want it here so that we can consume this script later and re-use
  if [[ $UPGRADE_CLUSTER == true ]]; then
    # End the prober test now before we start cluster upgrade, up until now we should have zero failed requests
    end_prober_test ${PROBER_PID} || return $?

    if [[ -n "$UPGRADE_OCP_IMAGE" ]]; then
      oc adm upgrade --to-image="${UPGRADE_OCP_IMAGE}" --force=true --allow-explicit-upgrade
      timeout 7200 '[[ $(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.image==\"${UPGRADE_OCP_IMAGE}\")].state}") != Completed ]]' || return 1
    else
      latest_cluster_version=$(oc adm upgrade | sed -ne '/VERSION/,$ p' | grep -v VERSION | awk '{print $1}' | sort -r | head -n 1)
      [[ $latest_cluster_version != "" ]] || return 1
      oc adm upgrade --to-latest=true --force=true
      timeout 7200 '[[ $(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.version==\"${latest_cluster_version}\")].state}") != Completed ]]' || return 1
    fi

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

  oc delete --ignore-not-found=true ksvc pizzaplanet-upgrade-service scale-to-zero-upgrade-service upgrade-probe -n serving-tests

  remove_temporary_gopath

  return 0
  )
}

function run_knative_serving_operator_tests {
  logger.info 'Running Serving operator tests'
  (
  local exitstatus=0
  checkout_knative_serving_operator

  export TEST_NAMESPACE="${SERVING_NAMESPACE}"

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2e \
    --kubeconfig "$KUBECONFIG" \
    || exitstatus=5$? && true

  print_test_result ${exitstatus}

  wait_for_knative_serving_ingress_ns_deleted || return 1

  remove_temporary_gopath

  return $exitstatus
  )
}
