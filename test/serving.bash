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
  cd "$KNATIVE_SERVING_HOME" || return $?

  prepare_knative_serving_tests || return $?

  # Enable allow-zero-initial-scale before running e2e tests (for test/e2e/initial_scale_test.go)
  oc -n ${KNATIVE_SERVING_VERSION} patch knativeserving/knative-serving --type=merge --patch='{"spec": {"config": { "autoscaler": {"allow-zero-initial-scale": "true"}}}}'

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
  local max_replicas min_replicas
  max_replicas=$(oc get hpa activator -n "$SERVING_NAMESPACE" -ojsonpath='{.spec.maxReplicas}')
  min_replicas=$(oc get hpa activator -n "$SERVING_NAMESPACE" -ojsonpath='{.spec.minReplicas}')

  # Keep this in sync with test/ha/ha.go
  readonly REPLICAS=2
  # TODO: Increase BUCKETS size more than 1 when operator supports configmap/config-leader-election setting.
  readonly BUCKETS=1

  # Changing the bucket count and cycling the controllers will leave around stale
  # lease resources at the old sharding factor, so clean these up.
  oc -n ${SERVING_NAMESPACE} delete leases --all

  # Wait for a new leader Controller to prevent race conditions during service reconciliation
  wait_for_leader_controller || failed=3

  # Dump the leases post-setup.
  oc get lease -n "${SERVING_NAMESPACE}"

  # Give the controller time to sync with the rest of the system components.
  sleep 30

  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch '{"spec": {"maxReplicas": '${REPLICAS}', "minReplicas": '${REPLICAS}'}}' || failed=4

  # Run HA tests separately as they're stopping core Knative Serving pods
  # Define short -spoofinterval to ensure frequent probing while stopping pods
  go_test_e2e -tags=e2e -timeout=15m -failfast -parallel=1 ./test/ha \
    -replicas="${REPLICAS}" -buckets="${BUCKETS}" -spoofinterval="10ms" \
    --resolvabledomain \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=5

  # Restore the original maxReplicas for any tests running after this test suite
  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch '{"spec": {"maxReplicas": '${max_replicas}', "minReplicas": '${min_replicas}'}}' || failed=6

  print_test_result ${failed}

  return $failed
  )
}

function run_knative_serving_rolling_upgrade_tests {
  logger.info "Running Serving rolling upgrade tests"
  (
  local failed upgrade_to latest_cluster_version cluster_version prev_serving_version latest_serving_version

  # Save the rootdir before changing dir
  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  cd "$KNATIVE_SERVING_HOME" || return $?

  prepare_knative_serving_tests || return $?

  failed=0
  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_SERVING_VERSION}:knative-serving-test-{{.Name}}"
  PROBE_FRACTION=1.0
  prev_serving_version=$(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}")

  if [[ ${prev_serving_version} < "0.14.0" ]]; then
    PROBE_FRACTION=0.95
  fi
  logger.info "Target success fraction is $PROBE_FRACTION"

  go_test_e2e -tags=preupgrade -timeout=20m ./test/upgrade \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain || return 1

  logger.info "Starting prober test"

  rm -f /tmp/prober-signal
  go_test_e2e -tags=probe -timeout=20m ./test/upgrade \
    -probe.success_fraction=$PROBE_FRACTION \
    --imagetemplate "$image_template" \
    --kubeconfig "$KUBECONFIG" \
    --resolvabledomain &

  # Wait for the upgrade-probe kservice to be ready before proceeding
  timeout 900 '[[ $(oc get services.serving.knative.dev upgrade-probe -n serving-tests -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]' || return 1

  PROBER_PID=$!

  if [[ $UPGRADE_SERVERLESS == true ]]; then
    latest_serving_version=$(echo $KNATIVE_SERVING_VERSION | sed "s/v//")

    logger.info "updating serving version from ${prev_serving_version} to ${latest_serving_version}"

    # Get latest CSV from the given channel
    upgrade_to="$CURRENT_CSV"

    cluster_version=$(oc get clusterversion -o=jsonpath="{.items[0].status.history[?(@.state==\"Completed\")].version}")
    if [[ "$cluster_version" = 4.1.* || "${HOSTNAME}" = *ocp-41* || \
          "$cluster_version" = 4.2.* || "${HOSTNAME}" = *ocp-42* ]]; then
      if approve_csv "$upgrade_to" "$OLM_UPGRADE_CHANNEL" ; then # Upgrade should fail on OCP 4.1, 4.2
        return 1
      fi
      # Check we got RequirementsNotMet error
      [[ $(oc get ClusterServiceVersion $upgrade_to -n $OPERATORS_NAMESPACE -o=jsonpath="{.status.requirementStatus[?(@.name==\"$upgrade_to\")].message}") =~ "requirement not met: minKubeVersion" ]] || return 1
      # Check KnativeServing still has the old version
      [[ $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") == "$prev_serving_version" ]] || return 1
    else
      approve_csv "$upgrade_to" "$OLM_UPGRADE_CHANNEL" || return 1
      # Check KnativeServing has the latest version with Ready status
      timeout 300 '[[ ! ( $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.version}") == $latest_serving_version && $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") == True ) ]]' || return 1
    fi
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

  return 0
  )
}
