#!/usr/bin/env bash

# disable SC2086(Double quote to prevent globbing and word splitting)
# as go_test_e2e wants to split OPENSHIFT_TEST_OPTIONS by space.
#
# shellcheck disable=SC2086

# For SC2164
set -e

function prepare_knative_serving_tests {
  logger.debug 'Preparing Serving tests'

  cd "$KNATIVE_SERVING_HOME"

  # Don't bother with the chaosduck downstream for now
  rm -fv test/config/chaosduck.yaml

  # workaround until https://github.com/knative/operator/issues/431 was fixed.
  rm -fv test/config/config-deployment.yaml

  # Create test resources (namespaces, configMaps, secrets)
  oc apply -f test/config/cluster-resources.yaml
  # Workaround for https://issues.redhat.com/browse/OSSM-1397
  if [[ $FULL_MESH == "true" ]]; then
    oc label namespace serving-tests maistra.io/member-of=istio-system --overwrite
  fi
  oc apply -f test/config/test-resources.yaml
  # Adding scc for anyuid to test TestShouldRunAsUserContainerDefault.
  oc adm policy add-scc-to-user anyuid -z default -n serving-tests
  # Add networkpolicy to test namespace and label to serving namespaces for testing under the strict networkpolicy.
  add_networkpolicy "serving-tests"
  add_networkpolicy "serving-tests-alt"

  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="${INGRESS_NAMESPACE}"
}

function upstream_knative_serving_e2e_and_conformance_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running Serving E2E and conformance tests"

  prepare_knative_serving_tests

  # Enable allow-zero-initial-scale before running e2e tests (for test/e2e/initial_scale_test.go)
  configure_cm autoscaler allow-zero-initial-scale:true

  # Enable ExternalIP for Kourier.
  oc -n "${SERVING_NAMESPACE}" patch knativeserving/knative-serving \
    --type=merge \
    --patch='{"spec": {"ingress": { "kourier": {"service-type": "LoadBalancer"}}}}'

  # Enable the required features for the respective tests.
  enable_feature_flags kubernetes.podspec-init-containers kubernetes.podspec-persistent-volume-claim \
  kubernetes.podspec-persistent-volume-write kubernetes.podspec-securitycontext

  # Create a persistent volume claim for the respective tests
  oc apply -f ./test/config/pvc/pvc.yaml

  # Apply resource quota in rq-test namespace, needed for the related e2e test.
  oc apply -f ./test/config/resource-quota/resource-quota.yaml

  image_template="registry.ci.openshift.org/openshift/knative-serving-test-{{.Name}}:${KNATIVE_SERVING_VERSION}"
  subdomain=$(oc get ingresses.config.openshift.io cluster  -o jsonpath="{.spec.domain}")
  OPENSHIFT_TEST_OPTIONS="--kubeconfig $KUBECONFIG --enable-beta --enable-alpha --resolvabledomain --customdomain=$subdomain --https --skip-cleanup-on-fail"

  local parallel=16
  if [[ $FULL_MESH == "true" ]]; then
    # TODO: SRVKS-211: Can not run grpc and http2 tests.
    rm ./test/e2e/grpc_test.go
    rm ./test/e2e/http2_test.go
    # Remove h2c test
    sed -ie '47,51d' ./test/conformance/runtime/protocol_test.go
    parallel=8
  fi

  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') = VSphere ]]; then
    # Since we don't have LoadBalancers working, gRPC tests will always fail.
    rm -f ./test/e2e/grpc_test.go
    parallel=2
  fi

  mv ./test/e2e/autoscale_test.go ./test/e2e/autoscale_test.backup

  SYSTEM_NAMESPACE="$SERVING_NAMESPACE" go_test_e2e -tags="e2e" -timeout=30m -parallel=$parallel \
    ./test/e2e ./test/conformance/api/... ./test/conformance/runtime/... \
    ./test/e2e/domainmapping \
    ./test/e2e/initcontainers \
    ./test/e2e/pvc \
    ${OPENSHIFT_TEST_OPTIONS} \
    --imagetemplate "$image_template"

  mv ./test/e2e/autoscale_test.backup ./test/e2e/autoscale_test.go
  # Run autoscale tests separately as they require more CPU resources
  SYSTEM_NAMESPACE="$SERVING_NAMESPACE" go_test_e2e -tags="e2e" -timeout=20m -parallel=3 \
    ./test/e2e \
    -run "TestAutoscale|TestRPSBased|TestTargetBurstCapacity|TestFastScaleToZero" \
    ${OPENSHIFT_TEST_OPTIONS} \
    --imagetemplate "$image_template"

  # Run the helloworld test with an image pulled into the internal registry.
  oc tag -n serving-tests "registry.ci.openshift.org/openshift/knative-serving-test-helloworld:${KNATIVE_SERVING_VERSION}" "helloworld:latest" --reference-policy=local
  SYSTEM_NAMESPACE="$SERVING_NAMESPACE" go_test_e2e -tags=e2e -timeout=30m ./test/e2e -run "^(TestHelloWorld)$" \
    ${OPENSHIFT_TEST_OPTIONS} \
    --imagetemplate "image-registry.openshift-image-registry.svc:5000/serving-tests/{{.Name}}"
  
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
  oc -n "${SERVING_NAMESPACE}" delete leases --all

  # Wait for a new leader Controller to prevent race conditions during service reconciliation
  wait_for_leader_controller

  # Dump the leases post-setup.
  oc get lease -n "${SERVING_NAMESPACE}"

  # Give the controller time to sync with the rest of the system components.
  sleep 30

  oc -n "$SERVING_NAMESPACE" patch hpa activator \
    --patch '{"spec": {"maxReplicas": '${REPLICAS}', "minReplicas": '${REPLICAS}'}}'

  # Run HA tests separately as they're stopping core Knative Serving pods
  # Define short -spoofinterval to ensure frequent probing while stopping pods
  SYSTEM_NAMESPACE="$SERVING_NAMESPACE" go_test_e2e -tags=e2e -timeout=15m -failfast -parallel=1 ./test/ha \
    -replicas="${REPLICAS}" -buckets="${BUCKETS}" -spoofinterval="10ms" \
    ${OPENSHIFT_TEST_OPTIONS} \
    --imagetemplate "$image_template"

  # Restore the original maxReplicas for any tests running after this test suite
  oc -n "$SERVING_NAMESPACE" patch hpa activator --patch \
    '{"spec": {"maxReplicas": '"${max_replicas}"', "minReplicas": '"${min_replicas}"'}}'

  ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')

  # Feature is tested on 4.11+ as this is the version we start enabling it by default.
  if versions.ge "$(versions.major_minor "$ocp_version")" "4.11"; then
      # Enable secure pod defaults for the following tests.
      enable_feature_flags secure-pod-defaults

    # Verify that the right sc is set by default at the revision side.
    go_test_e2e -timeout=10m -tags=e2e ./test/e2e/securedefaults -run "^(TestSecureDefaults)$" \
      ${OPENSHIFT_TEST_OPTIONS} \
      --imagetemplate "$image_template"

    # Allow to use any seccompProfile for non default cases,
    # for more check https://docs.openshift.com/container-platform/4.12/authentication/managing-security-context-constraints.html
    oc adm policy add-scc-to-user privileged -z default -n serving-tests

    # Verify that non secure settings are allowed, although not-recommended.
    # It requires scc privileged or a custom scc that allows any seccompProfile to be set.
    SYSTEM_NAMESPACE="$SERVING_NAMESPACE" go_test_e2e -tags=e2e -timeout=10m ./test/e2e/securedefaults -run "^(TestUnsafePermitted)$" \
      ${OPENSHIFT_TEST_OPTIONS} \
      --imagetemplate "$image_template"
  fi
}

function enable_feature_flags {
  for feature in "$@"; do
    echo "Enabling feature: $feature"
    configure_cm features "$feature":enabled
  done
  # Allow settings to be picked up
  sleep 30
}

function configure_cm {
  local cm="$1"
  local patch=""
  declare -A json_properties

  for property in "${@:2}"; do
    KEY="${property%%:*}"
    VALUE="${property##*:}"
    patch=${patch:+$patch,}"\"$KEY\": \"$VALUE\""
    # escape in case property contains dots eg. kubernetes.pod-spec
    j_property="$(echo "'$KEY'" | sed "s/\./\\\./g")"
    json_properties["$j_property"]="$VALUE"
  done

  oc -n ${SERVING_NAMESPACE} patch knativeserving/knative-serving --type=merge --patch="{\"spec\": {\"config\": { \"$cm\": {$patch} }}}"

  for j_property in "${!json_properties[@]}"; do
    timeout 30 "[[ ! \$(oc get cm -n ${SERVING_NAMESPACE} config-$cm -o jsonpath={.data.${j_property}}) == \"${json_properties[$j_property]}\" ]]"
  done
}
