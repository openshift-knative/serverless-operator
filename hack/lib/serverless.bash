#!/usr/bin/env bash

function ensure_serverless_installed {
  logger.info 'Check if Serverless is installed'
  local prev=${1:-false}
  if oc get knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1 && \
    oc get knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1 && \
    oc get knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1
  then
    logger.success 'Serverless is already installed.'
    return 0
  fi

  # Deploy config-logging configmap before running serving-opreator pod.
  # Otherwise, we cannot change log level by configmap.
  enable_debug_log

  if [[ $prev == "true" ]]; then
    install_serverless_previous
  else
    install_serverless_latest
  fi
}

function install_serverless_previous {
  logger.info "Installing previous version of Serverless..."

  # Remove installplan from previous installations, leaving this would make the operator
  # upgrade to the latest version immediately
  remove_installplan "$CURRENT_CSV"

  deploy_serverless_operator "$PREVIOUS_CSV"

  if [[ $INSTALL_SERVING == "true" ]]; then
    deploy_knativeserving_cr
  fi
  if [[ $INSTALL_EVENTING == "true" ]]; then
    deploy_knativeeventing_cr
  fi
  if [[ $INSTALL_KAFKA == "true" ]]; then
    deploy_knativekafka_cr
  fi

  logger.success "Previous version of Serverless is installed: $PREVIOUS_CSV"
}

function install_serverless_latest {
  logger.info "Installing latest version of Serverless..."
  deploy_serverless_operator_latest

  if [[ $INSTALL_SERVING == "true" ]]; then
    deploy_knativeserving_cr
  fi
  if [[ $INSTALL_EVENTING == "true" ]]; then
    deploy_knativeeventing_cr
  fi
  if [[ $INSTALL_KAFKA == "true" ]]; then
    deploy_knativekafka_cr
  fi

  logger.success "Latest version of Serverless is installed: $CURRENT_CSV"
}

function remove_installplan {
  local install_plan csv
  csv="${1:?Pass a CSV as arg[1]}"
  logger.info "Removing installplan for $csv"
  install_plan=$(find_install_plan "$csv")
  if [[ -n $install_plan ]]; then
    oc delete "$install_plan" -n "${OPERATORS_NAMESPACE}"
  else
    logger.debug "No install plan for $csv"
  fi
}

function deploy_serverless_operator_latest {
  deploy_serverless_operator "$CURRENT_CSV"
}

function deploy_serverless_operator {
  local csv tmpfile
  csv="${1:?Pass as CSV as arg[1]}"
  logger.info "Install the Serverless Operator: ${csv}"
  tmpfile=$(mktemp /tmp/subscription.XXXXXX.yaml)
  cat > "$tmpfile" <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: "${OPERATOR}"
  namespace: "${OPERATORS_NAMESPACE}"
spec:
  channel: "${OLM_CHANNEL}"
  name: "${OPERATOR}"
  source: "${OLM_SOURCE}"
  sourceNamespace: "${OLM_NAMESPACE}"
  installPlanApproval: Manual
  startingCSV: "${csv}"
EOF
  [ -n "$OPENSHIFT_CI" ] && cat "$tmpfile"
  oc apply -f "$tmpfile"

  # Approve the initial installplan automatically
  approve_csv "$csv" "$OLM_CHANNEL"
}

function approve_csv {
  local csv_version install_plan channel
  csv_version=${1:?Pass a CSV as arg[1]}
  channel=${2:?Pass channel as arg[2]}

  logger.info 'Ensure channel and source is set properly'
  oc patch subscriptions.operators.coreos.com "$OPERATOR" -n "${OPERATORS_NAMESPACE}" \
    --type 'merge' \
    --patch '{"spec": {"channel": "'"${channel}"'", "source": "'"${OLM_SOURCE}"'"}}'

  logger.info 'Wait for the installplan to be available'
  timeout 900 "[[ -z \$(find_install_plan ${csv_version}) ]]"

  install_plan=$(find_install_plan "${csv_version}")
  oc patch "$install_plan" -n "${OPERATORS_NAMESPACE}" \
    --type merge --patch '{"spec":{"approved":true}}'

  if ! timeout 300 "[[ \$(oc get ClusterServiceVersion $csv_version -n ${OPERATORS_NAMESPACE} -o jsonpath='{.status.phase}') != Succeeded ]]" ; then
    oc get ClusterServiceVersion "$csv_version" -n "${OPERATORS_NAMESPACE}" -o yaml || true
    return 105
  fi
}

function find_install_plan {
  local csv="${1:-Pass a CSV as arg[1]}"
  for plan in $(oc get installplan -n "${OPERATORS_NAMESPACE}" --no-headers -o name); do
    if [[ $(oc get "$plan" -n "${OPERATORS_NAMESPACE}" -o=jsonpath='{.spec.clusterServiceVersionNames}' | grep -c "$csv") -eq 1 && \
      $(oc get "$plan" -n "${OPERATORS_NAMESPACE}" -o=jsonpath="{.status.bundleLookups[0].catalogSourceRef.name}" | grep -c "$OLM_SOURCE") -eq 1 ]]
    then
      echo "$plan"
      return 0
    fi
  done
  echo ""
}

function deploy_knativeserving_cr {
  logger.info 'Deploy Knative Serving'
  local rootdir serving_cr

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeservings) -eq 0 ]]"

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  serving_cr="$(mktemp -t serving-XXXXX.yaml)"
  cp "${rootdir}/test/v1alpha1/resources/operator.knative.dev_v1alpha1_knativeserving_cr.yaml" "$serving_cr"

  if [[ $FULL_MESH == "true" ]]; then
    enable_istio "$serving_cr"
  fi

  if [[ $ENABLE_TRACING == "true" ]]; then
    enable_tracing "$serving_cr"
  fi

  oc apply -n "${SERVING_NAMESPACE}" -f "$serving_cr"

  if [[ $FULL_MESH == "true" ]]; then
    # metadata-webhook adds istio annotations for e2e test by webhook.
    oc apply -f "${rootdir}/serving/metadata-webhook/config"
  fi

  oc wait --for=condition=Ready knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" --timeout=900s

  logger.success 'Knative Serving has been installed successfully.'
}

# If ServiceMesh is enabled:
# - Set ingress.istio.enbled to "true"
# - Set inject and rewriteAppHTTPProbers annotations for activator and autoscaler
#   as "test/v1alpha1/resources/operator.knative.dev_v1alpha1_knativeserving_cr.yaml" has the value "prometheus".
function enable_istio {
  local custom_resource istio_patch
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  istio_patch="$(mktemp -t istio-XXXXX.yaml)"
  cat - << EOF > "${istio_patch}"
spec:
  ingress:
    istio:
      enabled: true
  deployments:
  - annotations:
      sidecar.istio.io/inject: "true"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: activator
  - annotations:
      sidecar.istio.io/inject: "true"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: autoscaler
EOF

  yq merge --inplace --arrays append "$custom_resource" "$istio_patch"

  rm -f "${istio_patch}"
}

function deploy_knativeeventing_cr {
  logger.info 'Deploy Knative Eventing'
  local rootdir eventing_cr

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeeventings.operator.knative.dev) -eq 0 ]]"

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  eventing_cr="$(mktemp -t serving-XXXXX.yaml)"

  # Install Knative Eventing
  cat <<EOF > "$eventing_cr"
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: ${EVENTING_NAMESPACE}
spec:
  config:
    logging:
      loglevel.controller: "debug"
      loglevel.webhook: "debug"
      loglevel.kafkachannel-dispatcher: "debug"
      loglevel.kafkachannel-controller: "debug"
      loglevel.inmemorychannel-dispatcher: "debug"
      loglevel.mt-broker-controller: "debug"
EOF

  if [[ $ENABLE_TRACING == "true" ]]; then
    enable_tracing "$eventing_cr"
  fi

  oc apply -f "$eventing_cr"

  oc wait --for=condition=Ready knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" --timeout=900s

  logger.success 'Knative Eventing has been installed successfully.'
}

function deploy_knativekafka_cr {
  logger.info 'Deploy Knative Kafka'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativekafkas.operator.serverless.openshift.io) -eq 0 ]]"

  # Install Knative Kafka
  cat <<EOF | oc apply -f -
apiVersion: operator.serverless.openshift.io/v1alpha1
kind: KnativeKafka
metadata:
  name: knative-kafka
  namespace: ${EVENTING_NAMESPACE}
spec:
  sink:
    enabled: true
  broker:
    enabled: true
    defaultConfig:
      bootstrapServers: my-cluster-kafka-bootstrap.kafka:9092
  source:
    enabled: true
  channel:
    enabled: true
    bootstrapServers: my-cluster-kafka-bootstrap.kafka:9092
EOF

  oc wait --for=condition=Ready knativekafkas.operator.serverless.openshift.io knative-kafka -n "$EVENTING_NAMESPACE" --timeout=900s

  logger.success 'Knative Kafka has been installed successfully.'
}

function ensure_kafka_channel_default {
  logger.info 'Set KafkaChannel as default'
  local defaultChConfig channelTemplateSpec yamls patchfile
  yamls="$(dirname "$(realpath "${BASH_SOURCE[0]}")")/yamls"
  defaultChConfig="$(cat "${yamls}/kafka-default-ch-config.yaml")"
  channelTemplateSpec="$(cat "${yamls}/kafka-channel-templatespec.yaml")"
  patchfile="$(mktemp -t kafka-dafault-XXXXX.json)"
  echo '{
  "spec": {
    "config": {
      "default-ch-webhook": {
        "default-ch-config": "'"${defaultChConfig//$'\n'/\\n}"'"
      },
      "config-br-default-channel": {
        "channel-template-spec": "'"${channelTemplateSpec//$'\n'/\\n}"'"
      }
    }
  }
}' > "${patchfile}"
  oc patch knativeeventing knative-eventing \
    -n "${EVENTING_NAMESPACE}" \
    --type merge --patch "$(cat "${patchfile}")"
  rm -f "${patchfile}"

  logger.success 'KafkaChannel is set as default.'
}

function teardown_serverless {
  logger.warn '😭  Teardown Serverless...'

  if oc get knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeServing CR'
    oc delete knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}"
  fi
  logger.info 'Ensure no knative serving pods running'
  timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${SERVING_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${SERVING_NAMESPACE}"
  fi
  logger.info 'Ensure no ingress pods running'
  timeout 600 "[[ \$(oc get pods -n ${INGRESS_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  timeout 600 "[[ \$(oc get ns ${INGRESS_NAMESPACE} --no-headers | wc -l) == 1 ]]"
  if oc get knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeEventing CR'
    oc delete knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}"
    # TODO: Remove workaround for stale pingsource resources (https://issues.redhat.com/browse/SRVKE-473)
    oc delete deployment -n "${EVENTING_NAMESPACE}" --ignore-not-found=true pingsource-mt-adapter
  fi
  # Order of deletion should not matter for Kafka and Eventing (SRVKE-667)
  # Try delete Kafka after Eventing
  if oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeKafka CR'
    oc delete knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}"
  fi
  logger.info 'Ensure no knative eventing or knative kafka pods running'
  timeout 600 "[[ \$(oc get pods -n ${EVENTING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${EVENTING_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${EVENTING_NAMESPACE}"
  fi
  logger.info 'Deleting subscription'
  oc delete subscriptions.operators.coreos.com \
    -n "${OPERATORS_NAMESPACE}" "${OPERATOR}" \
    --ignore-not-found
  logger.info 'Deleting ClusterServiceVersion'
  for csv in $(set +o pipefail && oc get csv -n "${OPERATORS_NAMESPACE}" --no-headers 2>/dev/null \
      | grep "${OPERATOR}" | cut -f1 -d' '); do
    oc delete csv -n "${OPERATORS_NAMESPACE}" "${csv}"
  done
  logger.info 'Ensure no operators present'
  timeout 600 "[[ \$(oc get deployments -n ${OPERATORS_NAMESPACE} -oname | grep -c 'knative') != 0 ]]"
  logger.info 'Deleting operators namespace'
  oc delete namespace "${OPERATORS_NAMESPACE}" --ignore-not-found=true
  logger.info 'Ensure not CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'knative.dev') -eq 0 ]]; then
    oc get crd -oname | grep 'knative.dev' | xargs oc delete --timeout=60s
  fi
  logger.success 'Serverless has been uninstalled.'
}

# Enable debug log on knative-serving-operator
function enable_debug_log {
cat <<-EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: config-logging
  namespace: ${OPERATORS_NAMESPACE}
data:
  zap-logger-config: |
    {
      "level": "debug",
      "development": false,
      "outputPaths": ["stdout"],
      "errorOutputPaths": ["stderr"],
      "encoding": "json",
      "encoderConfig": {
        "timeKey": "ts",
        "levelKey": "level",
        "nameKey": "logger",
        "callerKey": "caller",
        "messageKey": "msg",
        "stacktraceKey": "stacktrace",
        "lineEnding": "",
        "levelEncoder": "",
        "timeEncoder": "iso8601",
        "durationEncoder": "",
        "callerEncoder": ""
      }
    }
EOF
}

# == State dumps

function dump_state.setup {
  if (( INTERACTIVE )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi

  error_handlers.register dump_state
}

function dump_state {
  logger.info 'Dumping state...'
  logger.debug 'Environment variables:'
  env

  dump_subscriptions
  gather_knative_state
}

function dump_subscriptions {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces || true
}

function gather_knative_state {
  logger.info 'Gather knative state'
  local gather_dir="${ARTIFACT_DIR:-/tmp}/gather-knative"
  mkdir -p "$gather_dir"
  IMAGE_OPTION=("--image=quay.io/openshift-knative/must-gather")
  if [[ $FULL_MESH == true ]]; then
    IMAGE_OPTION=("${IMAGE_OPTION[@]}" "--image=registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel7")
  fi

  oc --insecure-skip-tls-verify adm must-gather \
    "${IMAGE_OPTION[@]}" \
    --dest-dir "$gather_dir" > "${gather_dir}/gather-knative.log"
}

# delete serverless test resources
function teardown_extras {
  logger.warn "😭  Teardown serverless extras..."

  # remove routes
  logger.info 'Removing serverless test routes'
  if oc get route -A | grep "metrics-eventing" >/dev/null 2>&1; then
    oc delete --ignore-not-found=true route/metrics-eventing -n openshift-serverless
  fi
  if oc get route -A | grep "metrics-serving" >/dev/null 2>&1; then
    oc delete --ignore-not-found=true route/metrics-serving -n openshift-serverless
  fi
  if oc get route myroute -n knative-serving-ingress >/dev/null 2>&1; then
    oc delete --ignore-not-found=true route/myroute -n knative-serving-ingress
  fi

  # remove csv and subscriptions
  logger.info 'Removing additional subscriptions and CSV'
  if oc get subscription.operators.coreos.com "${OPERATOR}" -n openshift-operators >/dev/null 2>&1; then
    CSV=$(oc get subscription.operators.coreos.com "${OPERATOR}" -n openshift-operators -o=custom-columns=CURRENT_CSV:.status.currentCSV --no-headers=true)
    oc delete --ignore-not-found=true clusterserviceversions.operators.coreos.com $CSV -n openshift-operators
    oc delete --ignore-not-found=true subscription.operators.coreos.com "${OPERATOR}" -n openshift-operators
  fi
  oc delete --ignore-not-found=true subscriptions.operators.coreos.com serverless-operator-subscription -n openshift-operators
  oc delete --ignore-not-found=true subscriptions.operators.coreos.com serverless-operator-subscription -n openshift-serverless

  # remove admission services
  logger.info 'Removing admission server services'
  oc delete --ignore-not-found=true service/admission-server-service -n openshift-operators
  oc delete --ignore-not-found=true service/admission-server-service -n openshift-serverless

  logger.success "Serverless extras have been removed."
}
