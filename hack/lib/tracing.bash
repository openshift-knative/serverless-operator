#!/usr/bin/env bash

function install_tracing {
  deploy_zipkin
  enable_eventing_tracing
}

function deploy_zipkin {
  logger.info "Installing Zipkin in namespace ${ZIPKIN_NAMESPACE}"
  cat <<EOF | oc apply -f -
apiVersion: v1
kind: Service
metadata:
  name: zipkin
  namespace: ${ZIPKIN_NAMESPACE}
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
  namespace: ${ZIPKIN_NAMESPACE}
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
        resources:
          limits:
            memory: 1000Mi
          requests:
            memory: 256Mi
---
EOF

  logger.info "Waiting until Zipkin is available"
  oc wait deployment --all --timeout=600s --for=condition=Available -n "${ZIPKIN_NAMESPACE}"
}

function enable_eventing_tracing {
  logger.info "Configuring tracing for Eventing"
  oc -n "${EVENTING_NAMESPACE}" patch knativeeventing/knative-eventing --type=merge \
    --patch='{"spec": {"config": { "tracing": {"enable":"true","backend":"zipkin", "zipkin-endpoint":"http://zipkin.'"${ZIPKIN_NAMESPACE}"'.svc.cluster.local:9411/api/v2/spans", "debug":"true", "sample-rate":"1.0"}}}}'
}

function teardown_tracing {
  logger.warn 'Teardown tracing'

  oc delete service    -n "${ZIPKIN_NAMESPACE}" zipkin --ignore-not-found
  oc delete deployment -n "${ZIPKIN_NAMESPACE}" zipkin --ignore-not-found

  timeout 600 "[[ \$(oc get pods -n ${ZIPKIN_NAMESPACE} --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"

  logger.success 'Tracing is uninstalled.'
}
