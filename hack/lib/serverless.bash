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
       $(oc get "$plan" -n "${OPERATORS_NAMESPACE}" -o=jsonpath="{.status.bundleLookups[0].catalogSourceRef.name}" | grep -c "$OLM_SOURCE") -eq 1 ]]; then
         echo "$plan"
         return 0
    fi
  done
  echo ""
}

function deploy_knativeserving_cr {
  logger.info 'Deploy Knative Serving'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeservings) -eq 0 ]]"

  local rootdir
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Install Knative Serving
  # Deploy the full version of KnativeServing (vs. minimal KnativeServing). The future releases should
  # ensure compatibility with this resource and its spec in the current format.
  # This is a way to test backwards compatibility of the product with the older full-blown configuration.
  oc apply -n "${SERVING_NAMESPACE}" -f "${rootdir}/test/v1alpha1/resources/operator.knative.dev_v1alpha1_knativeserving_cr.yaml"

  if [[ $FULL_MESH == "true" ]]; then
    enable_net_istio
  fi

  timeout 900 "[[ \$(oc get knativeserving.operator.knative.dev knative-serving \
    -n ${SERVING_NAMESPACE} -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != True ]]"

  logger.success 'Knative Serving has been installed successfully.'
}

# enable_net_istio adds patch to KnativeServing:
# - Set ingress.istio.enbled to "true"
# - Set inject and rewriteAppHTTPProbers annotations for activator and autoscaler
#   as "test/v1alpha1/resources/operator.knative.dev_v1alpha1_knativeserving_cr.yaml" has the value "prometheus".
function enable_net_istio {
  patchfile="$(mktemp -t knative-serving-XXXXX.yaml)"
  cat - << EOF > "${patchfile}"
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
  - name: domain-mapping
    replicas: 2
EOF

  oc patch knativeserving knative-serving \
    -n "${SERVING_NAMESPACE}" \
    --type merge --patch-file="${patchfile}"

  timeout 900 "[[ \$(oc get knativeserving.operator.knative.dev knative-serving \
    -n ${SERVING_NAMESPACE} -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != True ]]"

  logger.success 'KnativeServing has been updated successfully.'

  # metadata-webhook adds istio annotations for e2e test by webhook.
  oc apply -f https://raw.githubusercontent.com/nak3/metadata-webhook/main/examples/release.yaml
}

function deploy_knativeeventing_cr {
  logger.info 'Deploy Knative Eventing'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeeventings.operator.knative.dev) -eq 0 ]]"

  # Install Knative Eventing
  cat <<EOF | oc apply -f -
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: ${EVENTING_NAMESPACE}
spec:
  {}
EOF

  timeout 900 "[[ \$(oc get knativeeventing.operator.knative.dev \
    knative-eventing -n ${EVENTING_NAMESPACE} \
    -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != True ]]"

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
  source:
    enabled: true
  channel:
    enabled: true
    bootstrapServers: my-cluster-kafka-bootstrap.kafka:9092
EOF

  timeout 900 "[[ \$(oc get knativekafkas.operator.serverless.openshift.io \
    knative-kafka -n ${EVENTING_NAMESPACE} \
    -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != True ]]"

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
        "channelTemplateSpec": "'"${channelTemplateSpec//$'\n'/\\n}"'"
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


function ensure_kafka_no_auth {
  logger.info 'Ensure Knative Kafka using no Kafka auth'

  # Apply Knative Kafka
  cat <<EOF | oc apply -f - || return $?
apiVersion: operator.serverless.openshift.io/v1alpha1
kind: KnativeKafka
metadata:
  name: knative-kafka
  namespace: ${EVENTING_NAMESPACE}
spec:
  source:
    enabled: true
  channel:
    enabled: true
    bootstrapServers: my-cluster-kafka-bootstrap.kafka:9092
EOF

  # shellcheck disable=SC2016
  timeout 900 '[[ $(oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n $EVENTING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Kafka has been set to use no auth successfully.'
}

function ensure_kafka_tls_auth {
  logger.info 'Ensure Knative Kafka using TLS auth'

  # Apply Knative Kafka
  cat <<EOF | oc apply -f - || return $?
apiVersion: operator.serverless.openshift.io/v1alpha1
kind: KnativeKafka
metadata:
  name: knative-kafka
  namespace: ${EVENTING_NAMESPACE}
spec:
  source:
    enabled: true
  channel:
    enabled: true
    bootstrapServers: my-cluster-kafka-bootstrap.kafka:9093
    authSecretNamespace: default
    authSecretName: my-tls-secret
EOF

  # shellcheck disable=SC2016
  timeout 900 '[[ $(oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n $EVENTING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Kafka has been set to use TLS auth successfully.'
}

function ensure_kafka_sasl_auth {
  logger.info 'Ensure Knative Kafka using SASL auth'

  # Apply Knative Kafka
  cat <<EOF | oc apply -f - || return $?
apiVersion: operator.serverless.openshift.io/v1alpha1
kind: KnativeKafka
metadata:
  name: knative-kafka
  namespace: ${EVENTING_NAMESPACE}
spec:
  source:
    enabled: true
  channel:
    enabled: true
    bootstrapServers: my-cluster-kafka-bootstrap.kafka:9094
    authSecretNamespace: default
    authSecretName: my-sasl-secret
EOF

  # shellcheck disable=SC2016
  timeout 900 '[[ $(oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n $EVENTING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Kafka has been set to use SASL auth successfully.'
}

function teardown_serverless {
  logger.warn '😭  Teardown Serverless...'

  if oc get knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeServing CR'
    oc delete knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}"
  fi
  logger.info 'Ensure no knative serving pods running'
  timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    oc delete namespace "${SERVING_NAMESPACE}"
  fi
  logger.info 'Ensure no ingress pods running'
  timeout 600 "[[ \$(oc get pods -n ${INGRESS_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${INGRESS_NAMESPACE}" >/dev/null 2>&1; then
    oc delete namespace "${INGRESS_NAMESPACE}"
  fi
  # KnativeKafka must be deleted before KnativeEventing due to https://issues.redhat.com/browse/SRVKE-667
  if oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeKafka CR'
    oc delete knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}"
  fi
  if oc get knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeEventing CR'
    oc delete knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}"
    # TODO: Remove workaround for stale pingsource resources (https://issues.redhat.com/browse/SRVKE-473)
    oc delete deployment -n "${EVENTING_NAMESPACE}" --ignore-not-found=true pingsource-mt-adapter
  fi
  logger.info 'Ensure no knative eventing or knative kafka pods running'
  timeout 600 "[[ \$(oc get pods -n ${EVENTING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    oc delete namespace "${EVENTING_NAMESPACE}"
  fi

  oc delete subscriptions.operators.coreos.com \
    -n "${OPERATORS_NAMESPACE}" "${OPERATOR}" \
    --ignore-not-found
  for csv in $(set +o pipefail && oc get csv -n "${OPERATORS_NAMESPACE}" --no-headers 2>/dev/null \
      | grep serverless-operator | cut -f1 -d' '); do
    oc delete csv -n "${OPERATORS_NAMESPACE}" "${csv}"
  done
  oc delete namespace openshift-serverless --ignore-not-found=true

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
