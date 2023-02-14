#!/usr/bin/env bash

# == Overrides & test related

# shellcheck disable=SC1091,SC1090,SC2153
# See https://github.com/koalaman/shellcheck/issues/518
source "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")/hack/lib/__sources__.bash"

readonly TEARDOWN="${TEARDOWN:-on_exit}"
export TEST_NAMESPACE="${TEST_NAMESPACE:-serverless-tests}"
declare -a TEST_NAMESPACES
TEST_NAMESPACES=("${TEST_NAMESPACE}" "serverless-tests2" "serverless-tests-mesh")
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
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running operator e2e tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2e \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
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

  go_test_e2e -failfast -tags=e2e -timeout=30m -parallel=1 ./test/e2ekafka \
    --channel "$OLM_CHANNEL" \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
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

  if [[ $FULL_MESH == "true" ]]; then
    export GODEBUG="x509ignoreCN=0"
    go_test_e2e -failfast -timeout=60m -parallel=1 ./test/servinge2e/ \
      --kubeconfigs "${kubeconfigs_str}" \
      --imagetemplate "${IMAGE_TEMPLATE}" \
      "$@"
  else
    go_test_e2e -failfast -timeout=60m -parallel=1 ./test/servinge2e/... \
      --kubeconfigs "${kubeconfigs_str}" \
      --imagetemplate "${IMAGE_TEMPLATE}" \
      "$@"
  fi
}

function downstream_eventing_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Eventing downstream tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/eventinge2e \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
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

  # Used by eventing/test/lib
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/extensione2e/kafka \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_monitoring_e2e_tests {
  declare -a kubeconfigs
  local kubeconfigs_str

  logger.info "Running Knative monitoring tests"
  kubeconfigs+=("${KUBECONFIG}")
  for cfg in user*.kubeconfig; do
    kubeconfigs+=("$(pwd)/${cfg}")
  done
  kubeconfigs_str="$(array.join , "${kubeconfigs[@]}")"

  go_test_e2e -failfast -timeout=30m -parallel=1 ./test/monitoringe2e \
    --kubeconfigs "${kubeconfigs_str}" \
    --imagetemplate "${IMAGE_TEMPLATE}" \
    "$@"
}

function downstream_kitchensink_e2e_tests {
  logger.info "Running Knative kitchensink tests"

  # Create a secret for reconciler-test. The framework will copy this secret
  # to newly created namespaces and link to default service account in the namespace.
  if ! oc -n default get secret kn-test-image-pull-secret; then
    oc -n openshift-config get secret pull-secret -o yaml | \
      sed -e 's/name: .*/name: kn-test-image-pull-secret/' -e 's/namespace: .*/namespace: default/' | oc apply -f -
  fi

  # Used by the tests to get common ConfigMaps like config-logging
  SYSTEM_NAMESPACE="${SYSTEM_NAMESPACE:-"knative-eventing"}"
  export SYSTEM_NAMESPACE

  go_test_e2e -failfast -timeout=120m -parallel=8 ./test/kitchensinke2e \
  --imagetemplate "${IMAGE_TEMPLATE}" \
  "$@"
}

# == Upgrade testing

function run_rolling_upgrade_tests {
  logger.info "Running rolling upgrade tests"

  local base serving_image_version eventing_image_version eventing_kafka_image_version eventing_kafka_broker_image_version image_template channels common_opts

  serving_image_version=$(versions.major_minor "${KNATIVE_SERVING_VERSION}")
  eventing_image_version="${KNATIVE_EVENTING_VERSION}"
  eventing_kafka_image_version=$(versions.major_minor "${KNATIVE_EVENTING_KAFKA_VERSION}")
  eventing_kafka_broker_image_version="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION}"

  # mapping based on https://github.com/openshift/release/blob/master/core-services/image-mirroring/knative/mapping_knative_v1_2_quay
  base="quay.io/openshift-knative/"
  image_template=$(
    cat <<-EOF
$base{{- with .Name }}
{{- if eq .      "wathola-kafka-sender"}}{{.}}:v$eventing_kafka_image_version
{{- else if eq . "kafka-consumer"      }}knative-eventing-kafka-broker-test-kafka-consumer:$eventing_kafka_broker_image_version
{{- else if eq . "event-flaker"        }}knative-eventing-test-event-flaker:$eventing_image_version
{{- else if eq . "event-library"       }}knative-eventing-test-event-library:$eventing_image_version
{{- else if eq . "event-sender"        }}knative-eventing-test-event-sender:$eventing_image_version
{{- else if eq . "eventshub"           }}knative-eventing-test-eventshub:$eventing_image_version
{{- else if eq . "heartbeats"          }}knative-eventing-test-heartbeats:$eventing_image_version
{{- else if eq . "performance"         }}knative-eventing-test-performance:$eventing_image_version
{{- else if eq . "print"               }}knative-eventing-test-print:$eventing_image_version
{{- else if eq . "recordevents"        }}knative-eventing-test-recordevents:$eventing_image_version
{{- else if eq . "request-sender"      }}knative-eventing-test-request-sender:$eventing_image_version
{{- else if eq . "wathola-fetcher"     }}knative-eventing-test-wathola-fetcher:$eventing_image_version
{{- else if eq . "wathola-forwarder"   }}knative-eventing-test-wathola-forwarder:$eventing_image_version
{{- else if eq . "wathola-receiver"    }}knative-eventing-test-wathola-receiver:$eventing_image_version
{{- else if eq . "wathola-sender"      }}knative-eventing-test-wathola-sender:$eventing_image_version
{{- else                               }}{{.}}:v$serving_image_version{{end -}}
{{end -}}
EOF
)

  channels=messaging.knative.dev/v1beta1:KafkaChannel,messaging.knative.dev/v1:InMemoryChannel

  # Test configuration. See https://github.com/knative/eventing/tree/main/test/upgrade#probe-test-configuration
  # TODO(ksuszyns): remove EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO when knative/operator#297 is fixed.
  export EVENTING_UPGRADE_TESTS_SERVING_SCALETOZERO=false
  # Review this line as part of SRVCOM-2176
  export EVENTING_UPGRADE_TESTS_SERVING_USE=false
  export EVENTING_UPGRADE_TESTS_CONFIGMOUNTPOINT=/.config/wathola
  export EVENTING_UPGRADE_TESTS_TRACEEXPORTLIMIT=30
  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="${INGRESS_NAMESPACE}"
  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  common_opts=(./test/upgrade "-tags=upgrade" \
    "--kubeconfigs=${KUBECONFIG}" \
    "--channels=${channels}" \
    "--imagetemplate=${image_template}" \
    "--catalogsource=${OLM_SOURCE}" \
    "--upgradechannel=${OLM_UPGRADE_CHANNEL}" \
    "--csv=${CURRENT_CSV}" \
    "--csvprevious=${PREVIOUS_CSV}" \
    "--servingversion=${KNATIVE_SERVING_VERSION}" \
    "--eventingversion=${KNATIVE_EVENTING_VERSION/knative-v/}" \
    "--kafkaversion=${KNATIVE_EVENTING_KAFKA_BROKER_VERSION/knative-v/}" \
    "--servingversionprevious=${KNATIVE_SERVING_VERSION_PREVIOUS}" \
    "--eventingversionprevious=${KNATIVE_EVENTING_VERSION_PREVIOUS/knative-v/}" \
    "--kafkaversionprevious=${KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS/knative-v/}" \
    --resolvabledomain \
    --https)

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

function kitchensink_upgrade_tests {
  logger.info "Running kitchensink upgrade tests"

  local base image_template common_opts

  base="quay.io/openshift-knative/"
  image_template=$(
    cat <<-EOF
$base{{- with .Name }}
{{- if eq . "httpproxy" }}serving/{{.}}:v1.6
{{- else if eq . "recordevents" }}eventing/{{.}}:v1.6
{{- else if eq . "wathola-forwarder" }}eventing/{{.}}:v1.6
{{- else }}{{.}}:multiarch{{end -}}
{{end -}}
EOF
)

  export GATEWAY_OVERRIDE="kourier"
  export GATEWAY_NAMESPACE_OVERRIDE="${INGRESS_NAMESPACE}"
  export SYSTEM_NAMESPACE="$SERVING_NAMESPACE"

  common_opts=("-tags=upgrade" \
    "--kubeconfigs=${KUBECONFIG}" \
    "--imagetemplate=${image_template}" \
    "--catalogsource=${OLM_SOURCE}" \
    "--upgradechannel=${OLM_UPGRADE_CHANNEL}" \
    "--csv=${CURRENT_CSV}" \
    )

  go_test_e2e -run=TestKitchensink -timeout=90m ./test/upgrade/kitchensink "${common_opts[@]}"

  logger.success 'Upgrade kitchensink tests passed'
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
  curl -k -H "Authorization: Bearer $(oc -n openshift-monitoring sa get-token prometheus-k8s || oc -n openshift-monitoring create token prometheus-k8s)" \
    "https://${monitoring_route}/api/v1/alerts" | \
    jq -c '.data | map(select((.labels.service != "pingsource-mt-adapter") and (.labels.namespace == "'"${OPERATORS_NAMESPACE}"'" or .labels.namespace == "'"${EVENTING_NAMESPACE}"'" or .labels.namespace == "'"${SERVING_NAMESPACE}"'" or .labels.namespace == "'"${INGRESS_NAMESPACE}"'")))' > "${alerts_file}"

  num_alerts=$(jq 'length' "${alerts_file}")
  num_apiremoved_alerts=$(jq '.[].labels.alertname=="APIRemovedInNextEUSReleaseInUse-quick"' "${alerts_file}" | wc -l)
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
  local occmd num_users
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
    add_user "kubeadmin" "$(head -c 128 < /dev/urandom | base64 | fold -w 8 | head -n 1)"
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
