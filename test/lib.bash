#!/usr/bin/env bash

# == Overrides & test related

# shellcheck disable=SC1091,SC1090,SC2153
# See https://github.com/koalaman/shellcheck/issues/518
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
declare -a TEST_NAMESPACES
TEST_NAMESPACES=("${TEST_NAMESPACE}" "serverless-tests-mesh")
export TEST_NAMESPACES

source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/serving.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing.bash"
source "$(dirname "$(realpath "${BASH_SOURCE[0]}")")/eventing-kafka.bash"

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

# Overwritten, safe, version of test function from hack that acts well
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
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running operator e2e tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)

  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  RUN_FLAGS=(-failfast -timeout=30m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  go_test_e2e -tags=e2e "${RUN_FLAGS[@]}" ./test/e2e \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function serverless_operator_kafka_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Kafka tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  RUN_FLAGS=(-failfast -timeout=30m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  go_test_e2e -tags=e2e "${RUN_FLAGS[@]}" ./test/e2ekafka \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_serving_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Serving tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  RUN_FLAGS=(-failfast -timeout=60m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  if [[ "$USER_MANAGEMENT_ALLOWED" == "false" ]]; then
      mv ./test/servinge2e/user_permissions_test.go ./test/servinge2e/user_permissions_notest.go || true
  fi

  if [[ $MESH == "true" ]]; then
    go_test_e2e "${RUN_FLAGS[@]}" ./test/servinge2e/ ./test/servinge2e/servicemesh/ \
      --kubeconfigs "${kubeconfigs_str}" \
      --imagetemplate "${IMAGE_TEMPLATE}" \
      "$@"

    if [[ $(oc get Infrastructure cluster -ojson | jq .spec.platformSpec.type) == "\"AWS\"" ]]; then
      oc -n openshift-ingress-operator patch ingresscontroller/default --type=merge --patch='{"spec":{"endpointPublishingStrategy": {"type":"LoadBalancerService", "loadBalancer": {"scope":"External", "providerParameters":{"type":"AWS", "aws": {"type":"Classic", "classicLoadBalancer": {"connectionIdleTimeout":"20m"}}}}}}}'
      oc patch knativeservings.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" \
         --type 'merge' \
         --patch '{"spec": {"config": {"defaults": {"max-revision-timeout-seconds": "900", "revision-response-start-timeout-seconds": "700", "revision-timeout-seconds": "800"} }}}'

      go_test_e2e "${RUN_FLAGS[@]}" ./test/servinge2e/servicemesh/longrunning \
        --kubeconfigs "${kubeconfigs_str}" \
        --imagetemplate "${IMAGE_TEMPLATE}" \
        "$@"

      oc -n openshift-ingress-operator patch ingresscontroller/default --type=merge --patch='{"spec":{"endpointPublishingStrategy": {"type":"LoadBalancerService", "loadBalancer": {"scope":"External", "providerParameters":{"type":"AWS", "aws": {"type":"Classic", "classicLoadBalancer": {"connectionIdleTimeout":null}}}}}}}'
      oc patch knativeservings.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" \
         --type 'merge' \
         --patch '{"spec": {"config": {"defaults": {"max-revision-timeout-seconds": null, "revision-response-start-timeout-seconds": null, "revision-timeout-seconds": "300"} }}}'
    fi

  else
    go_test_e2e "${RUN_FLAGS[@]}" ./test/servinge2e/ ./test/servinge2e/kourier/ \
      --kubeconfigs "${kubeconfigs_str}" \
      --imagetemplate "${IMAGE_TEMPLATE}" \
      "$@"

    certName=$(oc get ingresscontroller.operator.openshift.io -n openshift-ingress-operator \
      default -o=jsonpath='{.spec.defaultCertificate.name}')
    if [[ "$certName" != "" ]]; then
      configure_cm network openshift-ingress-default-certificate:"${certName}"
    fi

    # Enable Serving encryption (only supported on Kourier - at least for now)
    configure_cm network system-internal-tls:enabled
    configure_cm network cluster-local-domain-tls:enabled

    logger.info "Restart controller to enable cert-manager integration"
    oc delete pod -n "${SERVING_NAMESPACE}" -l app=controller
    oc wait --timeout=60s --for=condition=Available deployment  -n "${SERVING_NAMESPACE}" controller

    logger.info "Restart activator to mount the certificates"
    oc delete pod -n "${SERVING_NAMESPACE}" -l app=activator
    oc wait --timeout=60s --for=condition=Available deployment  -n "${SERVING_NAMESPACE}" activator
    logger.info "cluster-local-domain-tls and system-internal-tls are ENABLED"

    go_test_e2e "${RUN_FLAGS[@]}" ./test/servinge2e/encryption/ \
      --kubeconfigs "${kubeconfigs_str}" \
      --imagetemplate "${IMAGE_TEMPLATE}" \
      "$@"

    # Put back default ingress certificate.
    if [[ "$certName" != "" ]]; then
      configure_cm network openshift-ingress-default-certificate:router-certs-default
    fi
    # Disable Serving encryption for following tests
    configure_cm network system-internal-tls:disabled
    configure_cm network cluster-local-domain-tls:disabled

    logger.info "Restart activator to unmount the certificates"
    oc delete pod -n "${SERVING_NAMESPACE}" -l app=activator
    oc wait --timeout=60s --for=condition=Available deployment  -n "${SERVING_NAMESPACE}" activator

    logger.info "Restart controller to disable cert-manager integration"
    oc delete pod -n "${SERVING_NAMESPACE}" -l app=controller
    oc wait --timeout=60s --for=condition=Available deployment  -n "${SERVING_NAMESPACE}" controller
    logger.info "cluster-local-domain-tls and system-internal-tls are DISABLED"
  fi
}

function downstream_eventing_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Eventing downstream tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  RUN_FLAGS=(-failfast -timeout=30m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  if [[ "$USER_MANAGEMENT_ALLOWED" == "false" ]]; then
      mv ./test/eventinge2e/user_permissions_test.go ./test/eventinge2e/user_permissions_notest.go || true
  fi

  go_test_e2e "${RUN_FLAGS[@]}" ./test/eventinge2e \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_eventing_e2e_rekt_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running Eventing REKT downstream tests"

  local images_file

  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  # Create a secret for reconciler-test. The framework will copy this secret
  # to newly created namespaces and link to default service account in the namespace.
  if ! oc -n default get secret kn-test-image-pull-secret; then
    oc -n openshift-config get secret pull-secret -o yaml | \
      sed -e 's/name: .*/name: kn-test-image-pull-secret/' -e 's/namespace: .*/namespace: default/' | oc apply -f -
  fi

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  RUN_FLAGS=(-failfast -timeout=30m -parallel=10)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  if [[ $MESH == "true" ]]; then
    # Need to specify a namespace that is in Mesh.
    go_test_e2e "${RUN_FLAGS[@]}" ./test/eventinge2erekt ./test/eventinge2erekt/servicemesh \
      --images.producer.file="${images_file}" \
      --poll.timeout=8m \
      --environment.namespace=serverless-tests \
      --istio.enabled="$MESH" \
      "$@"
  else
    go_test_e2e "${RUN_FLAGS[@]}" ./test/eventinge2erekt \
      --images.producer.file="${images_file}" \
      --poll.timeout=8m \
      "$@"
  fi
}

function downstream_knative_kafka_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Knative Kafka tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  RUN_FLAGS=(-failfast -timeout=30m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  if [[ "$USER_MANAGEMENT_ALLOWED" == "false" ]]; then
      mv ./test/extensione2e/kafka/user_permissions_test.go ./test/extensione2e/kafka/user_permissions_notest.go || true
  fi

  go_test_e2e "${RUN_FLAGS[@]}" ./test/extensione2e/kafka \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_knative_kafka_e2e_rekt_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running Knative Kafka REKT tests"

  local images_file

  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  # Create a secret for reconciler-test. The framework will copy this secret
  # to newly created namespaces and link to default service account in the namespace.
  if ! oc -n default get secret kn-test-image-pull-secret; then
    oc -n openshift-config get secret pull-secret -o yaml | \
      sed -e 's/name: .*/name: kn-test-image-pull-secret/' -e 's/namespace: .*/namespace: default/' | oc apply -f -
  fi

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  RUN_FLAGS=(-failfast -timeout=30m -parallel=10)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  if [[ $MESH == "true" ]]; then
    # Need to specify a namespace that is in Mesh.
    go_test_e2e "${RUN_FLAGS[@]}" ./test/extensione2erekt ./test/extensione2erekt/servicemesh \
      --images.producer.file="${images_file}" \
      --poll.timeout=8m \
      --environment.namespace=serverless-tests \
      --istio.enabled="$MESH" \
      "$@"

    # Workaround for https://github.com/knative-sandbox/eventing-kafka-broker/issues/3133
    oc delete secret strimzi-tls-secret -n serverless-tests || true
    oc delete secret strimzi-sasl-secret -n serverless-tests || true
  else
    go_test_e2e "${RUN_FLAGS[@]}" ./test/extensione2erekt \
      --images.producer.file="${images_file}" \
      --poll.timeout=8m \
      "$@"
  fi
}

function downstream_monitoring_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Knative monitoring tests"
  kubeconfigs+=("${KUBECONFIG}")
  while IFS= read -r -d '' cfg; do
    kubeconfigs+=("${cfg}")
  done < <(find "$(pwd -P)" -name 'user*.kubeconfig' -print0 | sort -z)
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  RUN_FLAGS=(-failfast -timeout=30m -parallel=1)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi

  go_test_e2e "${RUN_FLAGS[@]}" ./test/monitoringe2e \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_kitchensink_e2e_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running Knative kitchensink tests"

  local images_file workloads_patch

  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  # Create a secret for reconciler-test. The framework will copy this secret
  # to newly created namespaces and link to default service account in the namespace.
  if ! oc -n default get secret kn-test-image-pull-secret; then
    oc -n openshift-config get secret pull-secret -o yaml | \
      sed -e 's/name: .*/name: kn-test-image-pull-secret/' -e 's/namespace: .*/namespace: default/' | oc apply -f -
  fi

  # Patch dispatchers CPU requests to 300m for higher density (kitchensink tests do not generate high dataplane load)
  workloads_patch='{"spec": {"workloads": ['\
$( (for dispatcher in kafka-source-dispatcher kafka-channel-dispatcher kafka-broker-dispatcher; \
  do echo '{"name":"'"$dispatcher"'","resources":[{"container":"'"$dispatcher"'","requests":{"cpu":"300m"}}]}'; \
done;) | paste -sd "," - )']}}'

  oc patch knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" \
    --type 'merge' \
    --patch "$workloads_patch"

  # Used by the tests to get common ConfigMaps like config-logging
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  RUN_FLAGS=(-failfast -timeout=240m -parallel=8)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi
#  export GO_TEST_VERBOSITY=standard-verbose
  go_test_e2e "${RUN_FLAGS[@]}" ./test/kitchensinke2e \
  --images.producer.file="${images_file}" \
  --poll.timeout="8m" \
  --imagetemplate "${IMAGE_TEMPLATE}" \
  "$@"
}

function downstream_soak_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running soak tests"

  local images_file

  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  # Create a secret for reconciler-test. The framework will copy this secret
  # to newly created namespaces and link to default service account in the namespace.
  if ! oc -n default get secret kn-test-image-pull-secret; then
    oc -n openshift-config get secret pull-secret -o yaml | \
      sed -e 's/name: .*/name: kn-test-image-pull-secret/' -e 's/namespace: .*/namespace: default/' | oc apply -f -
  fi

  # Used by the tests to get common ConfigMaps like config-logging
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  # Patch kafka-source-dispatcher CPU requests to 300m for higher density (soak tests do not generate high dataplane load)
  oc patch knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" \
    --type 'merge' \
    --patch '{"spec": {"workloads": [{"name":"kafka-source-dispatcher","resources":[{"container":"kafka-source-dispatcher","requests":{"cpu":"300m"}}]}]}}'

  RUN_FLAGS=(-failfast -timeout=150m -parallel=512)
  if [ -n "${OPERATOR_TEST_FLAGS:-}" ]; then
    IFS=" " read -r -a RUN_FLAGS <<< "$OPERATOR_TEST_FLAGS"
  fi
#  export GO_TEST_VERBOSITY=standard-verbose
  go_test_e2e "${RUN_FLAGS[@]}" ./test/soak \
  --images.producer.file="${images_file}" \
  --soak-duration 120m --soak-copies 2 \
  "$@"
}

# == Upgrade testing

function run_rolling_upgrade_tests {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info "Running rolling upgrade tests"

  local image_template common_opts images_file

  # Specify image mapping for REKT tests
  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  default_knative_eventing_images
  default_knative_eventing_kafka_broker_images
  default_knative_serving_images

  # Mapping for non-REKT tests.
  image_template=$(
    cat <<-EOF
{{- with .Name }}
{{- if eq .      "kafka-consumer"      }}${KNATIVE_EVENTING_KAFKA_BROKER_TEST_KAFKA_CONSUMER}
{{- else if eq . "event-sender"        }}${KNATIVE_EVENTING_TEST_EVENT_SENDER}
{{- else if eq . "eventshub"           }}${KNATIVE_EVENTING_TEST_EVENTSHUB}
{{- else if eq . "heartbeats"          }}${KNATIVE_EVENTING_HEARTBEATS}
{{- else if eq . "print"               }}${KNATIVE_EVENTING_TEST_PRINT}
{{- else if eq . "recordevents"        }}${KNATIVE_EVENTING_TEST_RECORDEVENTS}
{{- else if eq . "request-sender"      }}${KNATIVE_EVENTING_TEST_REQUEST_SENDER}
{{- else if eq . "wathola-fetcher"     }}${KNATIVE_EVENTING_TEST_WATHOLA_FETCHER}
{{- else if eq . "wathola-forwarder"   }}${KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER}
{{- else if eq . "wathola-receiver"    }}${KNATIVE_EVENTING_TEST_WATHOLA_RECEIVER}
{{- else if eq . "wathola-sender"      }}${KNATIVE_EVENTING_TEST_WATHOLA_SENDER}
{{- else                               }}${KNATIVE_SERVING_IMAGE_PREFIX}-test-{{.}}:latest{{end -}}
{{end -}}
EOF
)

  echo "image template '$image_template'"

  # Test configuration. See https://github.com/knative/eventing/tree/main/test/upgrade#probe-test-configuration
  # TODO(ksuszyns): remove EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO when knative/operator#297 is fixed.
  export EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO=false
  # Review this line as part of SRVCOM-2176
  export EVENTING_UPGRADE_TESTS_SERVING_USE=false
  export EVENTING_UPGRADE_TESTS_CONFIGMOUNTPOINT=/.config/wathola
  export EVENTING_UPGRADE_TESTS_TRACEEXPORTLIMIT=30
  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  # There can be only one SYSTEM_NAMESPACE. Eventing and Serving tests both expect
  # some resources in their own system namespace. We copy the required resources from
  # EVENTING_NAMESPACE to SERVING_NAMESPACE and use that as system namespace.
  if ! oc -n "$SERVING_NAMESPACE" get configmap kafka-broker-config; then
    oc get configmap kafka-broker-config --namespace="$EVENTING_NAMESPACE" -o yaml | \
      sed -e 's/namespace: .*/namespace: '"$SERVING_NAMESPACE"'/' | \
      yq delete - metadata.ownerReferences | oc apply -f -
  fi

  common_opts=(-parallel=8 ./test/upgrade "-tags=upgrade" \
    "--kubeconfigs=${KUBECONFIG}" \
    "--imagetemplate=${image_template}" \
    "--images.producer.file=${images_file}" \
    "--catalogsource=${OLM_SOURCE}" \
    "--channel=${OLM_CHANNEL}" \
    "--upgradechannel=${OLM_UPGRADE_CHANNEL}" \
    "--csv=${CURRENT_CSV}" \
    "--csvprevious=${PREVIOUS_CSV}" \
    "--servingversion=${KNATIVE_SERVING_VERSION/knative-v/}" \
    "--eventingversion=${KNATIVE_EVENTING_VERSION/knative-v/}" \
    "--kafkaversion=${KNATIVE_EVENTING_KAFKA_BROKER_VERSION/knative-v/}" \
    "--servingversionprevious=${KNATIVE_SERVING_VERSION_PREVIOUS/knative-v/}" \
    "--eventingversionprevious=${KNATIVE_EVENTING_VERSION_PREVIOUS/knative-v/}" \
    "--kafkaversionprevious=${KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS/knative-v/}" \
    --resolvabledomain \
    --https)

  if [[ $MESH == "true" ]]; then
      common_opts+=("--environment.namespace=serverless-tests")
      common_opts+=("--istio.enabled")
      common_opts+=("--poll.timeout=8m")
      # For non-REKT eventing tests.
      common_opts+=("--reusenamespace")
  fi

  if [[ "${UPGRADE_SERVERLESS}" == "true" ]]; then
    # TODO: Remove creating the NS when this commit is backported: https://github.com/knative/serving/commit/1cc3a318e185926f5a408a8ec72371ba89167ee7
    if ! oc get namespace serving-tests &>/dev/null; then
      oc create namespace serving-tests
    fi
    # Run the two test suites one by one to prevent the situation when nested
    # tests time out and cause all other tests to have "Unknown" status.
    go_test_e2e -run=TestServerlessUpgradePrePost -timeout=90m "${common_opts[@]}"
    go_test_e2e -run=TestServerlessUpgradeContinual -timeout=60m "${common_opts[@]}"
  fi

  # For reuse in downstream test executions. Might be run after Serverless
  # upgrade or independently.
  if [[ "${UPGRADE_CLUSTER}" == "true" ]]; then
    if oc get namespace serving-tests &>/dev/null; then
      oc delete namespace serving-tests
    fi
    oc create namespace serving-tests
    # Make sure the cluster upgrade is run with latest version of Serverless as
    # the Serverless upgrade tests leave the product at the previous version (after downgrade).
    approve_csv "$CURRENT_CSV" "$OLM_UPGRADE_CHANNEL"
    go_test_e2e -run=TestClusterUpgrade -timeout=220m "${common_opts[@]}" \
      --openshiftimage="${UPGRADE_OCP_IMAGE}" \
      --upgradeopenshift
  fi

  # Delete the leftover namespace.
  oc delete namespace serving-tests

  logger.success 'Upgrade tests passed'
}

function kitchensink_csvs {
  local csvs csvs_rev csv_last csv_prefix csv
  # shellcheck disable=SC2034,SC2207
  csvs=( $(yq read --doc 0 "$rootdir/olm-catalog/serverless-operator-index/configs/index.yaml" 'entries[*].name') )

  array.reverse csvs csvs_rev
  # Remove first CSV as this is already installed.
  unset 'csvs_rev[0]'

  # Filter out .micro releases between .0 and the last .x, as they would be skipped due to skipVersion
  declare -a csvs_filtered
  csv_last="${csvs_rev[-1]}"
  csv_prefix=$(echo "$csv_last" | sed -E 's/\.[0-9]+$/./')
  for csv in "${csvs_rev[@]}"
  do
    if [ "${csv#"$csv_prefix"}" != "${csv}" ]
    then
      if [ "$csv" != "$csv_last" ] && [ "${csv#"$csv_prefix"}" != "0" ]; then
        continue
      fi
    fi
    csvs_filtered+=("$csv")
  done

  echo "${csvs_filtered[@]}" | tr ' ' ','
}

function kitchensink_upgrade_tests {
  logger.info "Running kitchensink upgrade tests"

  local images_file

  rootdir="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"

  images_file="${rootdir}/test/images-rekt.yaml"

  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  go_test_e2e -run=TestKitchensink -timeout=90m -parallel=20 ./test/upgrade/kitchensink -tags=upgrade \
     --kubeconfigs="${KUBECONFIG}" \
     --images.producer.file="${images_file}" \
     --imagetemplate="${IMAGE_TEMPLATE}" \
     --csv="$(kitchensink_csvs)" \
     --upgradechannel="${OLM_UPGRADE_CHANNEL}"

  logger.success 'Kitchensink upgrade tests passed'
}

function kitchensink_upgrade_stress_tests {
  logger.info "Running upgrade tests - stress control plane"

  local images_file

  images_file="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/images-rekt.yaml"

  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  go_test_e2e -run=TestUpgradeStress -timeout=90m -parallel=20 ./test/upgrade/kitchensink -tags=upgrade \
     --kubeconfigs="${KUBECONFIG}" \
     --images.producer.file="${images_file}" \
     --poll.timeout="8m" \
     --imagetemplate="${IMAGE_TEMPLATE}" \
     --catalogsource="${OLM_SOURCE}" \
     --upgradechannel="${OLM_UPGRADE_CHANNEL}" \
     --csv="${CURRENT_CSV}" \
     --servingversion="${KNATIVE_SERVING_VERSION/knative-v/}" \
     --eventingversion="${KNATIVE_EVENTING_VERSION/knative-v/}" \
     --kafkaversion="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION/knative-v/}"

  logger.success 'Upgrade tests - stress control plane - passed'
}

function teardown {
  if [ -n "$OPENSHIFT_CI" ]; then
    logger.warn 'Skipping teardown as we are running on Openshift CI'
    return 0
  fi
  logger.warn "Teardown 💀"
  teardown_serverless
  teardown_tracing
  # shellcheck disable=SC2153
  delete_namespaces "${SYSTEM_NAMESPACES[@]}" "${TEST_NAMESPACES[@]}"
  delete_catalog_source
  delete_users
}

function check_serverless_alerts {
  logger.info 'Checking Serverless alerts'
  local alerts_file monitoring_route num_alerts
  alerts_file="${ARTIFACTS:-/tmp}/alerts.json"
  monitoring_route=$(oc -n openshift-monitoring get routes alertmanager-main -oyaml -ojsonpath='{.spec.host}')
  # TODO(SRVKE-669) remove the filter for the pingsource-mt-adapter service once issue is fixed.
  curl -k -H "Authorization: Bearer $(oc -n openshift-monitoring create token prometheus-k8s)" \
    "https://${monitoring_route}/api/v2/alerts" | \
    jq -c 'map(select((.labels.service != "pingsource-mt-adapter") and (.labels.namespace == "'"${OPERATORS_NAMESPACE}"'" or .labels.namespace == "'"${EVENTING_NAMESPACE}"'" or .labels.namespace == "'"${SERVING_NAMESPACE}"'" or .labels.namespace == "'"${INGRESS_NAMESPACE}"'")))' > "${alerts_file}"

  num_alerts=$(jq 'length' "${alerts_file}")
  num_apiremoved_alerts=$(jq 'map(select(.labels.alertname=="APIRemovedInNextEUSReleaseInUse-quick")) | length' "${alerts_file}")
  if [ "${num_apiremoved_alerts}" = "${num_alerts}" ]; then
    echo -e "\n\nSkip APIRemovedInNextEUSReleaseInUse-quick alerts. Please see SRVCOM-1857 and bz2079314\n"
    return 0
  fi
  if [ ! "${num_alerts}" = "0" ]; then
    echo -e "\n\nERROR: Non-zero number of alerts: ${num_alerts}. Check ${alerts_file}\n"
    jq . "${alerts_file}"
    exit 1
  fi
}

function setup_quick_api_deprecation_alerts {
  local ocp_version
  ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
  # Setup deprecation alerts for OCP >= 4.8
  if versions.le "$(versions.major_minor "$ocp_version")" 4.7; then
    return
  fi
  logger.info "Setup quick API deprecation alerts"
  local namespaces=("${OPERATORS_NAMESPACE}" "${EVENTING_NAMESPACE}" "${SERVING_NAMESPACE}")
  if [[ "${SERVING_NAMESPACE}" != "${INGRESS_NAMESPACE}" ]]; then
    namespaces=("${namespaces[@]}" "${INGRESS_NAMESPACE}")
  fi
  for ns in "${namespaces[@]}"; do
    # Reuse the existing api-usage Prometheus rule and only make it react more quickly.
    oc get prometheusrule api-usage -n openshift-kube-apiserver -oyaml | \
      sed -e "s/\(.*name:.*\)/\1-quick/g" \
          -e "s/\(.*alert:.*\)/\1-quick/g" \
          -e "s/\(.*for:\).*/\1 1m/g" \
          -e "s/\(.*namespace:\).*/\1 ${ns}/g" | oc apply -f -
  done
}

# == Test users

function create_htpasswd_users {
  local num_users
  num_users=${num_users:-3}
  logger.info "Creating htpasswd for ${num_users} users"
  for i in $(seq 1 "$num_users"); do
    add_user "user${i}" "password${i}"
  done
  logger.success "${num_users} htpasswd users created"
}

function add_roles {
  logger.info "Adding roles to users"
  oc adm policy add-role-to-user admin user1 -n "$TEST_NAMESPACE"
  oc adm policy add-role-to-user edit user2 -n "$TEST_NAMESPACE"
  oc adm policy add-role-to-user view user3 -n "$TEST_NAMESPACE"
}

function ensure_kubeconfig {
  if [[ -z "$KUBECONFIG" ]]; then
    add_user "kubeadmin" "$(head -c 128 </dev/urandom | basenc --base64url | fold -w 8 | head -n 1)"
    oc adm policy add-cluster-role-to-user cluster-admin kubeadmin
    KUBECONFIG="$(pwd)/kubeadmin.kubeconfig"
    export KUBECONFIG
  fi
}

function delete_users {
  local user
  logger.info "Deleting users"
  while IFS= read -r line; do
    logger.debug "htpasswd user line: ${line}"
    user=$(echo "${line}" | cut -d: -f1)
    if [ -f "${user}.kubeconfig" ]; then
      rm -fv "${user}.kubeconfig"
    fi
  done < "users.htpasswd"
  rm -fv users.htpasswd
}

function add_networkpolicy {
  local NAMESPACE=${1:?Pass a namespace as arg[1]}
  cat <<EOF | oc apply -f -
---
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: deny-by-default
  namespace: "$NAMESPACE"
spec:
  podSelector:
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-from-system-namespace
  namespace: "$NAMESPACE"
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          knative.openshift.io/part-of: "openshift-serverless"
  podSelector: {}
  policyTypes:
  - Ingress
EOF
}

function wait_for_leader_controller() {
  local leader
  echo -n "Waiting for a leader Controller"
  for i in {1..150}; do  # timeout after 5 minutes
    local leader
    leader=$(set +o pipefail && oc get lease -n "${SERVING_NAMESPACE}" \
      -ojsonpath='{range .items[*].spec}{"\n"}{.holderIdentity}' \
      | cut -d'_' -f1 | grep "^controller-" | head -1)
    # Make sure the leader pod exists.
    if [ -n "${leader}" ] && oc get pod "${leader}" -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
      echo -e "\nNew leader Controller has been elected"
      return 0
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for leader controller"
  return 1
}

# Sets up a secret in the cert-manager namespace that contains the CA certs that need
# to be trusted to make TLS connections to routes of an arbitrary cluster.
# The Knative test machinery looks for this secret if the --https flag is engaged.
function trust_router_ca() {
  logger.info "Setting up cert-manager/ca-key-pair secret to trust router CA"

  # This is the secret the Knative test machinery looks for if the --https flag is engaged.
  certns="cert-manager"
  certname="ca-key-pair"

  certs=$(mktemp -d)
  oc -n openshift-config-managed get cm default-ingress-cert --template="{{index .data \"ca-bundle.crt\"}}" > "$certs/tls.crt"
  oc get ns $certns || oc create namespace $certns
  oc -n $certns get secret $certname || oc -n $certns create secret generic $certname --from-file=tls.crt="$certs/tls.crt"
}
