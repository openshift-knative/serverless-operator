#!/usr/bin/env bash

keda_resources_dir="$(dirname "${BASH_SOURCE[0]}")/keda_resources"

function install_custom_metrics_autoscaler_operator {
  logger.info "Installing KEDA operator"

  oc apply -f "${keda_resources_dir}"/subscription.yaml || return $?

  logger.info "Waiting until KEDA operator is available"
  timeout 600 "[[ \$(oc get deploy -n openshift-keda custom-metrics-autoscaler-operator --no-headers | wc -l) != 1 ]]" || return 1

  # Wait for the CRD we need to actually be active
  oc wait crd --timeout=600s kedacontrollers.keda.sh --for=condition=Established
}

function delete_custom_metrics_autoscaler_operator {
  logger.info "Deleting KEDA operator"

  oc delete -f "${keda_resources_dir}"/subscription.yaml --ignore-not-found || return $?

  logger.info "Waiting until KEDA operator is deleted"
  timeout 600 "[[ \$(oc get deploy -n openshift-keda custom-metrics-autoscaler-operator --no-headers | wc -l) != 0 ]]" || return 1
}

function install_keda_controller {

  oc apply -f "${keda_resources_dir}"/kedacontroller.yaml || return $?

  logger.info "Waiting until KEDA controllers are available"

  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-admission --no-headers | wc -l) != 1 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-metrics-apiserver --no-headers | wc -l) != 1 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-operator --no-headers | wc -l) != 1 ]]" || return 1
}

function delete_keda_controller {

  oc delete -f "${keda_resources_dir}"/kedacontroller.yaml --ignore-not-found  || return $?

  logger.info "Waiting until KEDA controllers are deleted"

  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-admission --no-headers | wc -l) != 0 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-metrics-apiserver --no-headers | wc -l) != 0 ]]" || return 1
  timeout 600 "[[ \$(oc get deploy -n openshift-keda keda-operator --no-headers | wc -l) != 0 ]]" || return 1
}

function install_keda {
  logger.info "KEDA install"
  ensure_catalog_pods_running
  install_custom_metrics_autoscaler_operator
  install_keda_controller
}

function uninstall_keda {
  logger.info "KEDA uninstall"
  delete_keda_controller
  delete_custom_metrics_autoscaler_operator
}
