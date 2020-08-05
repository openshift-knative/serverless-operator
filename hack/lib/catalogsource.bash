#!/usr/bin/env bash

function ensure_catalogsource_installed {
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"

  # HACK: Allow to run the image as root
  oc adm policy add-scc-to-user anyuid -z default -n openshift-marketplace

  oc -n openshift-marketplace new-build --binary --strategy=docker --name serverless-bundle
  oc -n openshift-marketplace start-build serverless-bundle --from-dir olm-catalog/serverless-operator -F

  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  ${rootdir}/hack/catalog.sh | oc apply -n "$OLM_NAMESPACE" -f - || return 1

  logger.success "CatalogSource installed successfully"
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource $OPERATOR"
  oc delete catalogsource --ignore-not-found=true -n "$OLM_NAMESPACE" "$OPERATOR" || return 10
  [ -f "$CATALOG_SOURCE_FILENAME" ] && rm -v "$CATALOG_SOURCE_FILENAME"
  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 900 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]" || return 11
  logger.success 'CatalogSource deleted'
}
