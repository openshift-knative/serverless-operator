#!/usr/bin/env bash

function install_tracing {
  if [[ "${TRACING_BACKEND}" == "zipkin" ]]; then
    install_zipkin_tracing
  else
    install_opentelemetry_tracing
  fi
}

function install_zipkin_tracing {
  logger.info "Installing Zipkin in namespace ${TRACING_NAMESPACE}"
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
            memory: 256Mi
---
EOF

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
  local target_namespace channel current_csv name
  name="${1:-Pass operator name as arg[1]}"
  logger.info "Install Operator: ${name}"
  target_namespace=openshift-operators
  channel=stable

  timeout 600 "[[ \$(oc get PackageManifest ${name} -n openshift-marketplace -o=custom-columns=DEFAULT_CHANNEL:.status.defaultChannel --no-headers=true) == '' ]]"
  current_csv=$(oc get packagemanifest "${name}" -n openshift-marketplace -o json | jq -r ".status.channels[] | select(.name == \"${channel}\") | .currentCSV")

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
  timeout 600 "[[ \$(oc get ClusterServiceVersion -n $target_namespace $current_csv -o jsonpath='{.status.phase}') != Succeeded ]]"
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
      jaeger:
        endpoint: jaeger-collector-headless.${TRACING_NAMESPACE}.svc:14250
        tls:
          ca_file: "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
      logging:
    service:
      pipelines:
        traces:
          receivers: [zipkin]
          processors: []
          exporters: [jaeger, logging]
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
EOF

  logger.info "Wait for Jaeger to be running"
  timeout 300 "[[ \$(oc get jaeger.jaegertracing.io jaeger -n ${TRACING_NAMESPACE} -o jsonpath='{.status.phase}') != Running ]]"
}

function enable_eventing_tracing {
  logger.info "Configuring tracing for Eventing"
  local endpoint
  endpoint=$(get_tracing_endpoint)
  oc -n "${EVENTING_NAMESPACE}" patch knativeeventing/knative-eventing --type=merge --patch='{"spec": {"config": { "tracing": {"enable":"true","backend":"zipkin", "zipkin-endpoint":"'"${endpoint}"'", "sample-rate":"'"${SAMPLE_RATE}"'"}}}}'
}

function enable_serving_tracing {
  logger.info "Configuring tracing for Serving"
  local endpoint
  endpoint=$(get_tracing_endpoint)
  oc -n "${SERVING_NAMESPACE}" patch knativeserving/knative-serving --type=merge --patch='{"spec": {"config": { "tracing": {"enable":"true","backend":"zipkin", "zipkin-endpoint":"'"${endpoint}"'", "sample-rate":"'"${SAMPLE_RATE}"'"}}}}'
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
  local csv

  # Teardown Zipkin
  oc delete service    -n "${TRACING_NAMESPACE}" zipkin --ignore-not-found
  oc delete deployment -n "${TRACING_NAMESPACE}" zipkin --ignore-not-found

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
