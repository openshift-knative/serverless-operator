#!/usr/bin/env bash

function install_service_mesh_operator {
  logger.info 'Install the ServiceMesh Operator'
  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: CatalogSourceConfig
metadata:
  name: ci-operators
  namespace: openshift-marketplace
spec:
  targetNamespace: openshift-operators
  packages: elasticsearch-operator,jaeger-product,kiali-ossm,servicemeshoperator
  source: redhat-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch-operator
  namespace: openshift-operators
spec:
  channel: preview
  name: elasticsearch-operator
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger-product
  namespace: openshift-operators
spec:
  channel: stable
  name: jaeger-product
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kiali-ossm
  namespace: openshift-operators
spec:
  channel: stable
  name: kiali-ossm
  source: ci-operators
  sourceNamespace: openshift-operators
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: servicemeshoperator
  namespace: openshift-operators
spec:
  channel: "1.0"
  name: servicemeshoperator
  source: ci-operators
  sourceNamespace: openshift-operators
EOF

  logger.info 'Wait for the istio-operator pod to appear'
  timeout 900 "[[ \$(oc get pods -n openshift-operators 2>/dev/null | grep -c istio-operator) -eq 0 ]]" || return 1

  logger.info 'Wait until the Operator pod is up and running'
  wait_until_pods_running openshift-operators || return 1

  logger.success "ServiceMesh Operator installed successfully"
}