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
  local rootdir
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Remove installplan from previous installations, leaving this would make the operator
  # upgrade to the latest version immediately
  remove_installplan "$CURRENT_CSV"

  deploy_serverless_operator "$PREVIOUS_CSV"  || return $?
  deploy_knativeserving_cr || return $?
}

function remove_installplan {
  local install_plan
  install_plan=$(find_install_plan $1)
  if [[ -n $install_plan ]]; then
    oc delete "$install_plan" -n "${OPERATORS_NAMESPACE}"
  fi
}

function install_serverless_latest {
  deploy_serverless_operator_latest || return $?

  if [[ $INSTALL_SERVING == "true" ]]; then
    deploy_knativeserving_cr || return $?
  fi
  if [[ $INSTALL_EVENTING == "true" ]]; then
    deploy_knativeeventing_cr || return $?
  fi
  if [[ $INSTALL_KAFKA == "true" ]]; then
    deploy_knativekafka_cr || return $?
  fi
}

function deploy_serverless_operator_latest {
  local rootdir
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  deploy_serverless_operator "$CURRENT_CSV"
}

function deploy_serverless_operator {
  local csv
  csv="$1"
  logger.info "Install the Serverless Operator ${csv}"

  cat <<EOF | oc apply -f - || return $?
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

  # Approve the initial installplan automatically
  approve_csv "$csv" "$OLM_CHANNEL" || return 5
}

function approve_csv {
  local csv_version install_plan channel
  csv_version=$1
  channel=$2

  # Ensure channel and source is set properly
  oc patch subscriptions.operators.coreos.com "$OPERATOR" -n "${OPERATORS_NAMESPACE}" \
    --type 'merge' \
    --patch '{"spec": {"channel": "'"${channel}"'", "source": "'"${OLM_SOURCE}"'"}}' \
    || return $?

  # Wait for the installplan to be available
  timeout 900 "[[ -z \$(find_install_plan $csv_version) ]]" || return 1

  install_plan=$(find_install_plan $csv_version)
  oc get $install_plan -n ${OPERATORS_NAMESPACE} -o yaml | sed 's/\(.*approved:\) false/\1 true/' | oc replace -f -

  if ! timeout 300 "[[ \$(oc get ClusterServiceVersion $csv_version -n ${OPERATORS_NAMESPACE} -o jsonpath='{.status.phase}') != Succeeded ]]" ; then
    oc get ClusterServiceVersion "$csv_version" -n "${OPERATORS_NAMESPACE}" -o yaml || true
    return 1
  fi
}

function find_install_plan {
  local csv=$1
  for plan in `oc get installplan -n ${OPERATORS_NAMESPACE} --no-headers -o name`; do 
    [[ $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath='{.spec.clusterServiceVersionNames}' | grep -c $csv) -eq 1 && \
       $(oc get $plan -n ${OPERATORS_NAMESPACE} -o=jsonpath="{.status.bundleLookups[0].catalogSourceRef.name}" | grep -c $OLM_SOURCE) -eq 1 ]] && echo $plan && return 0
  done
  echo ""
}

function deploy_knativeserving_cr {
  logger.info 'Deploy Knative Serving'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeservings) -eq 0 ]]" || return 6

  local rootdir
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Install Knative Serving
  # Deploy the full version of KnativeServing (vs. minimal KnativeServing). The future releases should
  # ensure compatibility with this resource and its spec in the current format.
  # This is a way to test backwards compatibility of the product with the older full-blown configuration.
  oc apply -n "${SERVING_NAMESPACE}" -f "${rootdir}/test/v1alpha1/resources/operator.knative.dev_v1alpha1_knativeserving_cr.yaml" || return $?

  timeout 900 '[[ $(oc get knativeserving.operator.knative.dev knative-serving -n $SERVING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Serving has been installed successfully.'
}

function deploy_knativeeventing_cr {
  logger.info 'Deploy Knative Eventing'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeeventings.operator.knative.dev) -eq 0 ]]" || return 6

  # Install Knative Eventing
  cat <<EOF | oc apply -f - || return $?
apiVersion: operator.knative.dev/v1alpha1
kind: KnativeEventing
metadata:
  name: knative-eventing
  namespace: ${EVENTING_NAMESPACE}
spec:
  {}
EOF

  timeout 900 '[[ $(oc get knativeeventing.operator.knative.dev knative-eventing -n $EVENTING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Eventing has been installed successfully.'
}

function deploy_knativekafka_cr {
  logger.info 'Deploy Knative Kafka'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativekafkas.operator.serverless.openshift.io) -eq 0 ]]" || return 6

  # Install Knative Kafka
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

  timeout 900 '[[ $(oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n $EVENTING_NAMESPACE -o=jsonpath="{.status.conditions[?(@.type==\"Ready\")].status}") != True ]]'  || return 7

  logger.success 'Knative Kafka has been installed sucessfully.'
}

function teardown_serverless {
  logger.warn '😭  Teardown Serverless...'

  if oc get knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeServing CR'
    oc delete knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" || return $?
  fi
  logger.info 'Ensure no knative serving pods running'
  timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]" || return 9

  if oc get knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeEventing CR'
    oc delete knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" || return $?
  fi
  if oc get knativekafkas.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeKafka CR'
    oc delete knativekafka.operator.serverless.openshift.io knative-kafka -n "${EVENTING_NAMESPACE}" || return $?
  fi
  logger.info 'Ensure no knative eventing or knative kafka pods running'
  timeout 600 "[[ \$(oc get pods -n ${EVENTING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]" || return 10

  oc delete subscriptions.operators.coreos.com -n "${OPERATORS_NAMESPACE}" "${OPERATOR}" 2>/dev/null
  for ip in $(oc get installplan -n "${OPERATORS_NAMESPACE}" | grep serverless-operator | cut -f1 -d' '); do
    oc delete installplan -n "${OPERATORS_NAMESPACE}" $ip
  done
  for csv in $(oc get csv -n "${OPERATORS_NAMESPACE}" | grep serverless-operator | cut -f1 -d' '); do
    oc delete csv -n "${OPERATORS_NAMESPACE}" "${csv}"
  done
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
