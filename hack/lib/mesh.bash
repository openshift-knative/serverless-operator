#!/usr/bin/env bash
  
mesh_deployments=(istio-operator jaeger-operator kiali-operator)

function install_mesh {
  deploy_servicemesh_operators
}

function uninstall_mesh {
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  logger.info "Installing service mesh operators in namespace openshift-operators"
  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: jaeger-product
  namespace: openshift-operators
spec:
  channel: stable
  name: jaeger-product
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kiali-ossm
  namespace: openshift-operators
spec:
  channel: stable
  name: kiali-ossm
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: servicemeshoperator
  namespace: openshift-operators
spec:
  channel: stable
  name: servicemeshoperator
  installPlanApproval: Automatic
  source: redhat-operators
  sourceNamespace: openshift-marketplace
EOF

  logger.info "Waiting until service mesh operators are available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators ${mesh_deployments[*]} --no-headers | wc -l) != 3 ]]" || return 1
  oc wait --for=condition=Available deployment "${mesh_deployments[@]}" --timeout=300s -n openshift-operators || return $?
}

function undeploy_servicemesh_operators {
  logger.info "Deleting service mesh subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator kiali-ossm jaeger-product --ignore-not-found
}

