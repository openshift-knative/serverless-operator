#!/usr/bin/env bash

function ensure_catalogsource_installed {
  logger.info 'Check if CatalogSource is installed'
  if oc get catalogsource serverless-operator -n "$OPERATORS_NAMESPACE" >/dev/null 2>&1; then
    logger.success 'CatalogSource is already installed.'
    return 0
  fi
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"

  local operator_image rootdir
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  operator_image=$(tag_operator_image)
  if [[ -n "${operator_image}" ]]; then
    "${rootdir}/hack/catalog.sh" | sed -e "s+\(.* containerImage:\)\(.*\)+\1 ${operator_image}+g" > "$CATALOG_SOURCE_FILENAME"
  else
    "${rootdir}/hack/catalog.sh" > "$CATALOG_SOURCE_FILENAME"
  fi
  oc apply -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME" || return 1

  logger.success "CatalogSource installed successfully"
}

function tag_operator_image {
  if [[ -n "${OPENSHIFT_BUILD_NAMESPACE:-}" ]]; then
    oc policy add-role-to-group system:image-puller "system:serviceaccounts:${OPERATORS_NAMESPACE}" --namespace="${OPENSHIFT_BUILD_NAMESPACE}" >/dev/null
    oc tag --insecure=false -n "${OPERATORS_NAMESPACE}" "${OPENSHIFT_REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:${OPERATOR} ${OPERATOR}:latest" >/dev/null
    echo "$INTERNAL_REGISTRY/$OPERATORS_NAMESPACE/$OPERATOR"
  fi
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource"
  if [ ! -f "$CATALOG_SOURCE_FILENAME" ]; then
    logger.success 'CatalogSource already deleted'
    return 0
  fi
  oc delete --ignore-not-found=true -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME" || return 10
  rm -v "$CATALOG_SOURCE_FILENAME"

  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 900 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]" || return 11

  logger.success 'CatalogSource deleted'
}
