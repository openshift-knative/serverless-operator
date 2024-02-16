#!/usr/bin/env bash

function install_tracing {
  if [[ "${TRACING_BACKEND}" == "zipkin" ]]; then
    if [[ "$ZIPKIN_DEDICATED_NODE" == "true" ]]; then
      dedicate_node_to_zipkin
    fi
    install_zipkin_tracing
  else
    install_opentelemetry_tracing
  fi
}

function dedicate_node_to_zipkin {
  logger.info "Placing zipkin taint on first worker node"
  local zipkin_node
  if [[ -z "$(oc get node -l 'zipkin,node-role.kubernetes.io/worker')"  ]]; then
    zipkin_node=$(oc get node -l 'node-role.kubernetes.io/worker' -ojsonpath='{.items[0].metadata.name}')
    # Add label for placing the Zipkin pod via nodeAffinity
    oc label node "$zipkin_node" zipkin=
    # Add taint to prevent pods other than Zipkin from scheduling there
    oc adm taint --overwrite=true node "$zipkin_node" zipkin:NoSchedule
  fi
}

function install_zipkin_tracing {
  logger.info "Installing Zipkin in namespace ${TRACING_NAMESPACE}"
  local ocp_version nodeAffinity=""
  local memory_requests=${ZIPKIN_MEMORY_REQUESTS:-"256Mi"}
  if [[ "$ZIPKIN_DEDICATED_NODE" == "true" ]]; then
  nodeAffinity=$(cat <<-EOF
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: zipkin
                operator: Exists
EOF
)
  fi

  cat <<EOF | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  name: zipkin
  namespace: ${TRACING_NAMESPACE}
spec:
  type: NodePort
  ports:
  - name: http
    port: 9411
  selector:
    app: zipkin
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zipkin
  namespace: ${TRACING_NAMESPACE}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: zipkin
  template:
    metadata:
      labels:
        app: zipkin
      annotations:
        sidecar.istio.io/inject: "false"
    spec:
      securityContext:
        runAsNonRoot: true
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: zipkin
        image: ghcr.io/openzipkin/zipkin:2
        ports:
        - containerPort: 9411
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: metadata.namespace
        - name: JAVA_OPTS
          value: '-Xms128m -Xmx9G -XX:+ExitOnOutOfMemoryError'
        - name: MEM_MAX_SPANS
          value: '10000000'
        resources:
          limits:
            memory: 10Gi
          requests:
            memory: ${memory_requests}
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
      tolerations:
      - key: zipkin
        operator: Exists
        effect: NoSchedule
${nodeAffinity}
---
EOF

  # Remove incompatible part for OCP 4.10 and older
  ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
  if versions.le "$(versions.major_minor "$ocp_version")" 4.10; then
    oc patch -n "${TRACING_NAMESPACE}" deployment zipkin --type='json' \
      -p "[{'op':'remove','path':'/spec/template/spec/securityContext/seccompProfile'}]"
  fi

  logger.info "Waiting until Zipkin is available"
  oc wait deployment --all --timeout=600s --for=condition=Available -n "${TRACING_NAMESPACE}"
}

function install_opentelemetry_tracing {
  logger.info "Install OpenTelemetry Tracing"
  if [[ $(oc get crd servicemeshcontrolplanes.maistra.io --no-headers | wc -l) != 1 ]]; then
    # The following components are installed with Service Mesh.
    logger.info "Install Distributed Tracing Platform (Jaeger) Operator"
    install_jaeger_operator
    install_jaeger_cr
  fi
  logger.info "Install Distributed Tracing Data Collection Operator"
  install_opentelemetry_operator
  install_opentelemetrycollector
}

function install_jaeger_operator {
  install_operator "jaeger-product"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators jaeger-operator --no-headers | wc -l) != 1 ]]"
  oc wait --for=condition=Available deployment jaeger-operator --timeout=300s -n openshift-operators
}

function install_opentelemetry_operator {
  install_operator "opentelemetry-product"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators opentelemetry-operator-controller-manager --no-headers | wc -l) != 1 ]]"
  oc wait --for=condition=Available deployment opentelemetry-operator-controller-manager --timeout=300s -n openshift-operators
}

function install_operator {
  local target_namespace channel current_csv name packagemanifest
  name="${1:-Pass operator name as arg[1]}"
  logger.info "Install Operator: ${name}"
  packagemanifest=$(mktemp /tmp/packagemanifest.XXXXXX.json)
  target_namespace=openshift-operators
  channel=stable

  timeout 600 "! oc get PackageManifest ${name} -n openshift-marketplace -ojson > ${packagemanifest}"
  current_csv=$(jq -r ".status.channels[] | select(.name == \"${channel}\") | .currentCSV" "${packagemanifest}")

  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${name}
  namespace: ${target_namespace}
spec:
  channel: ${channel}
  installPlanApproval: Automatic
  name: ${name}
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: "${current_csv}"
EOF

  logger.info "Waiting for CSV $current_csv to Succeed"
  wait_for_csv_succeeded "${current_csv}" "${name}" "${target_namespace}"
}

function wait_for_csv_succeeded {
  local seconds timeout csv ns subscription subscription_error

  timeout=600
  csv="${1:?Pass csv name as arg[1]}"
  subscription="${2:?Pass subscription name as arg[2]}"
  ns="${3:?Pass namespace as arg[3]}"
  interval="${interval:-1}"

  seconds=0
  restarts=0
  ln=' ' logger.debug "${*} : Waiting until non-zero (max ${timeout} sec.)"
  while (eval "[[ \$(oc get ClusterServiceVersion -n $ns $csv -o jsonpath='{.status.phase}') != Succeeded ]]" 2>/dev/null); do
    # Make sure there are .status.conditions available before parsing via jq
    oc wait --for=condition=CatalogSourcesUnhealthy=False subscription.operators.coreos.com "${subscription}" -n "${ns}" --timeout=120s
    subscription_error=$(oc get subscription.operators.coreos.com "${subscription}" -n "${ns}" -ojson | jq '.status.conditions[] | select(.message != null) | select(.message|test("exists and is not referenced by a subscription"))')
    if [[ "${subscription_error}" != "" && $restarts -lt 3 ]]; then
      logger.warn "Restarting OLM pods to work around OCPBUGS-19046"
      oc delete pods -n openshift-operator-lifecycle-manager -l app=catalog-operator
      oc delete pods -n openshift-operator-lifecycle-manager -l app=olm-operator
      restarts=$(( restarts + 1 ))
    fi
    seconds=$(( seconds + interval ))
    echo -n '.'
    sleep "$interval"
    [[ $seconds -gt $timeout ]] && echo '' \
      && logger.error "Time out of ${timeout} exceeded" \
      && return 71
  done
  [[ $seconds -gt 0 ]] && echo -n ' '
  echo 'done'
  return 0
}

function install_opentelemetrycollector {
  logger.info "Install OpenTelemetryCollector CR"
  # Workaround for TBD
  timeout 30 "! apply_opentelemetry_cr"
  logger.info "Wait for collector deployment to be available"
  timeout 600 "[[ \$(oc get deployment -n ${TRACING_NAMESPACE} cluster-collector-collector --no-headers | wc -l) != 1 ]]"
  oc wait --for=condition=Available deployment cluster-collector-collector --timeout=300s -n "${TRACING_NAMESPACE}"
}

function apply_opentelemetry_cr {
  cat <<EOF | oc apply -f - || return 1
apiVersion: opentelemetry.io/v1alpha1
kind: OpenTelemetryCollector
metadata:
  name: cluster-collector
  namespace: ${TRACING_NAMESPACE}
spec:
  mode: deployment
  config: |
    receivers:
      zipkin:
    processors:
    exporters:
      otlp:
        endpoint: jaeger-collector-headless.${TRACING_NAMESPACE}.svc:4317
        tls:
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
      logging:
    service:
      pipelines:
        traces:
          receivers: [zipkin]
          processors: []
          exporters: [otlp, logging]
EOF
}

function install_jaeger_cr {
  logger.info "Install Jaeger CR"

  cat <<EOF | oc apply -f -
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: jaeger
  namespace: ${TRACING_NAMESPACE}
spec:
  strategy: allInOne
  allInOne:
    options:
      collector:
        otlp:
          enabled: true
          grpc:
            tls:
              enabled: true
              cert: /etc/tls-config/tls.crt
              key: /etc/tls-config/tls.key
EOF

  logger.info "Wait for Jaeger to be running"
  timeout 300 "[[ \$(oc get jaeger.jaegertracing.io jaeger -n ${TRACING_NAMESPACE} -o jsonpath='{.status.phase}') != Running ]]"
}

function enable_tracing {
  local custom_resource tracing_endpoint tracing_patch
  custom_resource=${1:?Pass a custom resource to be patched as arg[1]}

  tracing_endpoint=$(get_tracing_endpoint)
  tracing_patch="$(mktemp -t tracing-XXXXX.yaml)"
  cat - << EOF > "$tracing_patch"
spec:
  config:
    tracing:
      backend: zipkin
      debug: "false"
      enable: "true"
      sample-rate: "${SAMPLE_RATE}"
      zipkin-endpoint: "${tracing_endpoint}"
EOF

  yq merge --inplace --arrays=append --overwrite "$custom_resource" "$tracing_patch"

  rm -f "${tracing_patch}"
}

function get_tracing_endpoint {
  if [[ "${TRACING_BACKEND}" == "zipkin" ]]; then
    echo "http://zipkin.${TRACING_NAMESPACE}.svc.cluster.local:9411/api/v2/spans"
  else
    echo "http://cluster-collector-collector-headless.${TRACING_NAMESPACE}.svc:9411/api/v2/spans"
  fi
}

function teardown_tracing {
  logger.warn 'Teardown Tracing'
  local csv zipkin_node

  # Teardown Zipkin
  oc delete service    -n "${TRACING_NAMESPACE}" zipkin --ignore-not-found
  oc delete deployment -n "${TRACING_NAMESPACE}" zipkin --ignore-not-found

  if [[ -n "$(oc get node -l 'zipkin,node-role.kubernetes.io/worker')"  ]]; then
    zipkin_node=$(oc get node -l 'zipkin,node-role.kubernetes.io/worker' -ojsonpath='{.items[0].metadata.name}')
    oc label node "$zipkin_node" zipkin-
    oc adm taint node "$zipkin_node" zipkin:NoSchedule-
  fi

  # Teardown OpenTelemetry
  if oc get -n "${TRACING_NAMESPACE}" opentelemetrycollector.opentelemetry.io cluster-collector &>/dev/null; then
    oc delete -n "${TRACING_NAMESPACE}" opentelemetrycollector.opentelemetry.io cluster-collector
    timeout 600 "[[ \$(oc get -n ${TRACING_NAMESPACE} deployment cluster-collector-collector --no-headers | wc -l) != 0 ]]"
  fi

  oc delete -n openshift-operators subscriptions.operators.coreos.com opentelemetry-product --ignore-not-found
  if oc get csv -n openshift-operators -oname | grep opentelemetry-operator &>/dev/null; then
    csv=$(oc get csv -n openshift-operators -oname | grep opentelemetry-operator)
    oc delete -n openshift-operators "${csv}" --ignore-not-found
  fi

  # Do not remove Jaeger if it's part of Service Mesh
  if [[ $(oc get crd servicemeshcontrolplanes.maistra.io --no-headers | wc -l) != 1 ]]; then
    if [[ $(oc get -n "${TRACING_NAMESPACE}" jaeger.jaegertracing.io jaeger --no-headers | wc -l) != 0 ]]; then
      oc delete -n "${TRACING_NAMESPACE}" jaeger.jaegertracing.io jaeger --ignore-not-found
      timeout 600 "[[ \$(oc get -n ${TRACING_NAMESPACE} deployment jaeger --no-headers | wc -l) != 0 ]]"
    fi
    oc delete -n openshift-operators subscriptions.operators.coreos.com jaeger-product --ignore-not-found
    if oc get csv -n openshift-operators -oname | grep jaeger-operator &>/dev/null; then
      csv=$(oc get csv -n openshift-operators -oname | grep jaeger-operator)
      oc delete -n openshift-operators "${csv}" --ignore-not-found
    fi
  fi

  timeout 600 "[[ \$(oc get pods -n ${TRACING_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"

  logger.success 'Tracing is uninstalled.'
}
