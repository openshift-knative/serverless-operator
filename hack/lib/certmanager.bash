#!/usr/bin/env bash

certmanager_resources_dir="$(dirname "${BASH_SOURCE[0]}")/certmanager_resources"
  
function install_certmanager {
  deploy_certmanager_operator
}

function uninstall_certmanager {
  undeploy_certmanager_operator
}

function deploy_certmanager_operator {
  logger.info "Installing cert manager operator"
  oc apply -f "${certmanager_resources_dir}"/subscription.yaml || return $?

  logger.info "Waiting until cert manager operator is available"

  timeout 600 "[[ \$(oc get deploy -n cert-manager cert-manager --no-headers | wc -l) != 1 ]]" || return 1
}

function undeploy_certmanager_operator {
  logger.info "Deleting cert manager subscriptions"
  oc delete -f "${certmanager_resources_dir}"/subscription.yaml || return $?

  logger.info 'Ensure no CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'maistra.io') -eq 0 ]]; then
    oc get crd -oname | grep 'cert-manager.io' | xargs oc delete --timeout=60s
  fi
  logger.success "cert manager has been uninstalled"
}
