#!/usr/bin/env bash


function install_cluster_logging {
  logger.info "Install Cluster Logging"
  ensure_catalog_pods_running
  install_namespace_rbac
  install_elasticsearch_operator
  install_clusterlogging_operator
  create_clusterlogging_cr
}

function install_elasticsearch_operator {
  logger.info "Install ElasticSearch operator"
  local target_namespace channel current_csv
  target_namespace=openshift-operators

  logger.info "Parse ElasticSearch default channel"
  timeout 600 "[[ \$(oc get PackageManifest elasticsearch-operator -n openshift-marketplace -o=custom-columns=DEFAULT_CHANNEL:.status.defaultChannel --no-headers=true) == '' ]]"
  channel=$(oc get PackageManifest elasticsearch-operator -n openshift-marketplace -o=custom-columns=DEFAULT_CHANNEL:.status.defaultChannel --no-headers=true)
  current_csv=$(oc get packagemanifest elasticsearch-operator -n openshift-marketplace -o json | jq -r ".status.channels[] | select(.name == \"${channel}\") | .currentCSV")

  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: elasticsearch-operator
  namespace: openshift-operators
spec:
  channel: "${channel}"
  installPlanApproval: Automatic
  name: elasticsearch-operator
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: "${current_csv}"
EOF

  logger.info "Waiting for CSV $current_csv to Succeed"
  timeout 600 "[[ \$(oc get ClusterServiceVersion -n openshift-operators $current_csv -o jsonpath='{.status.phase}') != Succeeded ]]"
}

function install_clusterlogging_operator {
  logger.info "Install ClusterLogging operator"
  local target_namespace channel current_csv
  target_namespace=openshift-logging
  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: "${target_namespace}-operatorgroup"
  namespace: "${target_namespace}"
spec:
  targetNamespaces:
  - "${target_namespace}"
EOF

  logger.info "Parse ClusterLogging default channel"
  timeout 600 "[[ \$(oc get PackageManifest cluster-logging -n openshift-marketplace -o=custom-columns=DEFAULT_CHANNEL:.status.defaultChannel --no-headers=true) == '' ]]"
  channel=$(oc get PackageManifest cluster-logging -n openshift-marketplace -o=custom-columns=DEFAULT_CHANNEL:.status.defaultChannel --no-headers=true)
  current_csv=$(oc get packagemanifest cluster-logging -n openshift-marketplace -o json | jq -r ".status.channels[] | select(.name == \"${channel}\") | .currentCSV")

  cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: cluster-logging
  namespace: ${target_namespace}
spec:
  channel: "${channel}"
  installPlanApproval: Automatic
  name: cluster-logging
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: "${current_csv}"
EOF

  logger.info "Waiting for CSV $current_csv to Succeed"
  timeout 600 "[[ \$(oc get ClusterServiceVersion -n $target_namespace $current_csv -o jsonpath='{.status.phase}') != Succeeded ]]"
}

function install_namespace_rbac {
  logger.info "Install namespaces and RBAC"
  local logging_namespace=openshift-logging
  local elasticsearch_namespace=openshift-operators-redhat

  cat <<EOF | oc apply -f -
apiVersion: v1
kind: Namespace
metadata:
  name: "${logging_namespace}"
  annotations:
    openshift.io/node-selector: ""
  labels:
    openshift.io/cluster-logging: "true"
    openshift.io/cluster-monitoring: "true"
---
apiVersion: v1
kind: Namespace
metadata:
  name: "${elasticsearch_namespace}"
  annotations:
    openshift.io/node-selector: ""
  labels:
    openshift.io/cluster-logging: "true"
    openshift.io/cluster-monitoring: "true"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: prometheus-k8s
  namespace: "${elasticsearch_namespace}"
rules:
- apiGroups:
  - ""
  resources:
  - services
  - endpoints
  - pods
  verbs:
  - get
  - list
  - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: prometheus-k8s
  namespace: "${elasticsearch_namespace}"
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: prometheus-k8s
subjects:
- kind: ServiceAccount
  name: prometheus-k8s
namespace: "${elasticsearch_namespace}"
EOF
}

function create_clusterlogging_cr {
  logger.info "Create ClusterLogging CR"
  local logging_namespace=openshift-logging
  cat <<EOF | oc apply -f -
apiVersion: "logging.openshift.io/v1"
kind: "ClusterLogging"
metadata:
  name: "instance"
  namespace: "${logging_namespace}"
spec:
  managementState: "Managed"
  logStore:
    type: "elasticsearch"
    elasticsearch:
      nodeCount: 1
      resources:
        limits:
          cpu: "4"
          memory: "4Gi"
        requests:
          cpu: "100m"
          memory: "1Gi"
      storage: {} # use emptyDir ephemeral storage
      redundancyPolicy: "ZeroRedundancy"
  visualization:
    type: "kibana"
    kibana:
      replicas: 1
  curation:
    type: "curator"
    curator:
      schedule: "30 3 * * *"
  collection:
    logs:
      type: "fluentd"
      fluentd: {}
EOF

  logger.info "Wait for pods other than cluster-logging operator to appear"
  timeout 600 "[[ \$(oc get pods -n $logging_namespace --no-headers | grep -v -c cluster-logging-operator) -eq 0 ]]"
}
