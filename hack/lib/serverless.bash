#!/usr/bin/env bash

function ensure_serverless_installed {
  logger.info 'Check if Serverless is installed'
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

  local csv
  if [[ "${INSTALL_OLDEST_COMPATIBLE}" == "true" ]]; then
    rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
    csv=$(yq read --doc 0 "$rootdir/olm-catalog/serverless-operator/index/configs/index.yaml" 'entries[-1].name')
  elif [[ "${INSTALL_PREVIOUS_VERSION}" == "true" ]]; then
    csv="$PREVIOUS_CSV"
  else
    csv="$CURRENT_CSV"
  fi

  # Remove installplan from previous installations, leaving this would make the operator
  # upgrade to the latest version immediately
  if [[ "$csv" != "$CURRENT_CSV" ]]; then
    remove_installplan "$CURRENT_CSV"
  fi

   if [[ ${SKIP_OPERATOR_SUBSCRIPTION:-} != "true" ]]; then
    logger.info "Installing Serverless version $csv"
    deploy_serverless_operator "$csv"
  fi

  install_knative_resources "${csv#serverless-operator.v}"

  logger.success "Serverless is installed: $csv"
}

function install_knative_resources {
  local serverless_version
  serverless_version=${1:?Pass serverless version as arg[1]}

  # Deploy the resources first and let them install in parallel, then
  # wait for them all to be ready.
  if [[ $INSTALL_SERVING == "true" ]]; then
    deploy_knativeserving_cr "$serverless_version"
  fi
  if [[ $INSTALL_EVENTING == "true" ]]; then
    deploy_knativeeventing_cr
  fi

  if [[ $INSTALL_SERVING == "true" ]]; then
    wait_for_knative_serving_ready
  fi
  if [[ $INSTALL_EVENTING == "true" ]]; then
    wait_for_knative_eventing_ready
  fi

  # https://issues.redhat.com/browse/SRVKE-1415 KnativeEventing a prerequisite to KnativeKafka
  if [[ $INSTALL_KAFKA == "true" ]]; then
    deploy_knativekafka_cr
  fi
  if [[ $INSTALL_KAFKA == "true" ]]; then
    wait_for_knative_kafka_ready
  fi
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
  local rootdir serving_cr serverless_version
  serverless_version=${1:?Pass serverless version as arg[1]}

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativeservings) -eq 0 ]]"

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  serving_cr="$(mktemp -t serving-XXXXX.yaml)"
  cp "${rootdir}/test/v1beta1/resources/operator.knative.dev_v1beta1_knativeserving_cr.yaml" "$serving_cr"

  if [[ "$serverless_version" != "${CURRENT_CSV#serverless-operator.v}" ]]; then
    logger.warn "Disabling internal encryption in upgrade tests due to SRVKS-1107."
    yq delete --inplace "$serving_cr" spec.config.network.internal-encryption
  fi

  if [[ $MESH == "true" ]]; then
    enable_istio "$serving_cr"
  fi

  if [[ $ENABLE_TRACING == "true" ]]; then
    enable_tracing "$serving_cr"
  fi

  if [[ $HA == "false" ]]; then
    yq write --inplace "$serving_cr" spec.high-availability.replicas 1
  fi

  if [[ "" != $(oc get ingresscontroller default -n openshift-ingress-operator -ojsonpath='{.spec.defaultCertificate}') ]]; then
    override_ingress_cert "$serving_cr"
  fi

  if [[ $USE_RELEASE_NEXT == "true" ]]; then
    # Apply the same change as in https://github.com/openshift-knative/serving/pull/608
    yq delete --inplace "$serving_cr" spec.config.network.internal-encryption
  fi

  oc apply -n "${SERVING_NAMESPACE}" -f "$serving_cr"

  if [[ $MESH == "true" ]]; then
    # metadata-webhook adds istio annotations for e2e test by webhook.
    oc apply -f "${rootdir}/serving/metadata-webhook/config"
  fi
}

# If ServiceMesh is enabled:
# - Set ingress.istio.enbled to "true"
# - Set inject and rewriteAppHTTPProbers annotations for activator and autoscaler
#   as "test/v1beta1/resources/operator.knative.dev_v1beta1_knativeserving_cr.yaml" has the value "prometheus".
function enable_istio {
  local custom_resource istio_patch
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  istio_patch="$(mktemp -t istio-XXXXX.yaml)"
  cat - << EOF > "${istio_patch}"
metadata:
  annotations:
    serverless.openshift.io/disable-istio-net-policies-generation: "true"
spec:
  ingress:
    istio:
      enabled: true
  config:
    istio: # point these to our own specific gateways now
      gateway.knative-serving.knative-ingress-gateway: knative-istio-ingressgateway.knative-serving-ingress.svc.cluster.local
      local-gateway.knative-serving.knative-local-gateway: knative-local-gateway.knative-serving-ingress.svc.cluster.local
  deployments:
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: activator
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: autoscaler
EOF

  yq merge --inplace --arrays append "$custom_resource" "$istio_patch"

  rm -f "${istio_patch}"
}

# If ServiceMesh is enabled:
# - Set ingress.istio.enabled to "true"
# - Set inject and rewriteAppHTTPProbers annotations for activator and autoscaler
#   as "test/v1beta1/resources/operator.knative.dev_v1beta1_knativeserving_cr.yaml" has the value "prometheus".
function enable_istio_eventing {
  local custom_resource istio_patch
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  istio_patch="$(mktemp -t istio-XXXXX.yaml)"
  cat - << EOF > "${istio_patch}"
spec:
  config:
    config-features:
      istio: "enabled"
      delivery-timeout: "enabled"
  workloads:
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: pingsource-mt-adapter
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: mt-broker-ingress
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: mt-broker-filter
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: imc-dispatcher
EOF

  yq merge --inplace --arrays append "$custom_resource" "$istio_patch"

  rm -f "${istio_patch}"
}

function override_ingress_cert {
  local custom_resource network_patch cert_name
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  cert_name=$(oc get ingresscontroller default -n openshift-ingress-operator \
    -ojsonpath='{.spec.defaultCertificate.name}')

  network_patch="$(mktemp -t network-XXXXX.yaml)"
  cat - << EOF > "${network_patch}"
spec:
  config:
    network:
      openshift-ingress-default-certificate: "${cert_name}"
EOF

  yq merge --inplace --arrays append "$custom_resource" "$network_patch"

  rm -f "${network_patch}"
}

# If ServiceMesh is enabled:
# - Set ingress.istio.enabled to "true"
# - Set inject and rewriteAppHTTPProbers annotations for activator and autoscaler
#   as "test/v1beta1/resources/operator.knative.dev_v1beta1_knativeserving_cr.yaml" has the value "prometheus".
function enable_istio_eventing_kafka {
  local custom_resource istio_patch
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  istio_patch="$(mktemp -t istio-XXXXX.yaml)"
  cat - << EOF > "${istio_patch}"
spec:
  workloads:
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-broker-receiver
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-broker-dispatcher
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-channel-receiver
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-channel-dispatcher
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-sink-receiver
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-source-dispatcher
  - labels:
      sidecar.istio.io/inject: "true"
    annotations:
      sidecar.istio.io/logLevel: "debug"
      sidecar.istio.io/rewriteAppHTTPProbers: "true"
    name: kafka-controller
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
  eventing_cr="$(mktemp -t eventing-XXXXX.yaml)"
  cp "${rootdir}/test/v1beta1/resources/operator.knative.dev_v1beta1_knativeeventing_cr.yaml" "$eventing_cr"

  if [[ $ENABLE_TRACING == "true" ]]; then
    enable_tracing "$eventing_cr"
  fi
  if [[ $MESH == "true" ]]; then
    enable_istio_eventing "$eventing_cr"
  fi

  if [[ $HA == "false" ]]; then
    yq write --inplace "$eventing_cr" spec.high-availability.replicas 1
  fi

  oc apply -n "${EVENTING_NAMESPACE}" -f "$eventing_cr"
}

function deploy_knativekafka_cr {
  logger.info 'Deploy Knative Kafka'

  # Wait for the CRD to appear
  timeout 900 "[[ \$(oc get crd | grep -c knativekafkas.operator.serverless.openshift.io) -eq 0 ]]"

  knativekafka_cr="$(mktemp -t knativekafka-XXXXX.yaml)"

  # Install Knative Kafka
  cat <<EOF > "$knativekafka_cr"
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

  if [[ $MESH == "true" ]]; then
    enable_istio_eventing_kafka "$knativekafka_cr"
  fi

  if [[ $HA == "false" ]]; then
    yq write --inplace "$knativekafka_cr" spec.high-availability.replicas 1
  fi

  if [[ $ENABLE_KEDA == "true" ]]; then
    yq write --inplace "$knativekafka_cr" 'spec.config.kafka-features."controller-autoscaler-keda"' "enabled"
  fi


  oc apply -f "$knativekafka_cr"
}

function wait_for_knative_serving_ready {
  oc wait --for=condition=Ready knativeserving.operator.knative.dev knative-serving -n "${SERVING_NAMESPACE}" --timeout=900s
  logger.success 'Knative Serving has been installed successfully.'
}

function wait_for_knative_eventing_ready {
  oc wait --for=condition=Ready knativeeventing.operator.knative.dev knative-eventing -n "${EVENTING_NAMESPACE}" --timeout=900s
  logger.success 'Knative Eventing has been installed successfully.'
}

function wait_for_knative_kafka_ready {
  oc wait --for=condition=Ready knativekafkas.operator.serverless.openshift.io knative-kafka -n "$EVENTING_NAMESPACE" --timeout=15m
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
  logger.info 'Ensure no knative serving pods running'
  timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${SERVING_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${SERVING_NAMESPACE}"
  fi
  logger.info 'Ensure no ingress pods running'
  timeout 600 "[[ \$(oc get pods -n ${INGRESS_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
  if oc get namespace "${INGRESS_NAMESPACE}" &>/dev/null; then
    oc delete namespace "${INGRESS_NAMESPACE}"
  fi
  timeout 600 "[[ \$(oc get ns ${INGRESS_NAMESPACE} --no-headers | wc -l) == 1 ]]"
  logger.info 'Ensure no knative eventing or knative kafka pods running'
  timeout 700 "[[ \$(oc get pods -n ${EVENTING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
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
  if [[ "${DELETE_CRD_ON_TEARDOWN}" == "true" ]]; then
    if [[ ! $(oc get crd -oname | grep -c 'knative.dev') -eq 0 ]]; then
      oc get crd -oname | grep 'knative.dev' | xargs oc delete --timeout=60s
    fi
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
  local gatherImageKnative="${MUST_GATHER_IMAGE_KNATIVE:-quay.io/openshift-knative/must-gather}"
  local gatherImageMesh="${MUST_GATHER_IMAGE_MESH:-registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel7}"
  mkdir -p "$gather_dir"
  IMAGE_OPTION=("--image=${gatherImageKnative}")
  if [[ $MESH == true ]]; then
    IMAGE_OPTION=("${IMAGE_OPTION[@]}" "--image=${gatherImageMesh}")
  fi

  oc --insecure-skip-tls-verify adm must-gather \
    "${IMAGE_OPTION[@]}" \
    --dest-dir "$gather_dir" > "${gather_dir}/gather-knative.log"
}
