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

  openshift_version=$(oc version -o yaml | yq read - openshiftVersion)
  deployment_namespace="cert-manager"
  if printf '%s\n4.12\n' "${openshift_version}" | sort --version-sort -C; then
      # OCP version is older as 4.12 and thus cert-manager-operator is only available as tech-preview in this version (cert-manager-operator GA'ed in OCP 4.12)
      
      echo "Running on OpenShift ${openshift_version} which supports cert-manager-operator only in tech-preview"

      yq --inplace 'del(select(document_index == 1) | .spec)' "${certmanager_resources_dir}"/subscription.yaml | \
      yq --inplace 'select(document_index == 2) | .spec.channel = "tech-preview"' "${certmanager_resources_dir}"/subscription.yaml | \
      oc apply -f - || return $?

      deployment_namespace="openshift-cert-manager"
  else
    echo "Running on OpenShift ${openshift_version} which supports GA'ed cert-manager-operator"

    oc apply -f "${certmanager_resources_dir}"/subscription.yaml || return $?
  fi

  logger.info "Waiting until cert manager operator is available"
  timeout 600 "[[ \$(oc get deploy -n ${deployment_namespace} cert-manager --no-headers | wc -l) != 1 ]]" || return 1
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
