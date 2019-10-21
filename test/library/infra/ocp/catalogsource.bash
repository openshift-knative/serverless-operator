#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash

function install_catalogsource {
  logger.info "Installing CatalogSource"

  local operator_image
  operator_image=$(tag_operator_image)
  if [[ -n "${operator_image}" ]]; then
    ./hack/catalog.sh | sed -e "s+\(.* containerImage:\)\(.*\)+\1 ${operator_image}+g" > "$CATALOG_SOURCE_FILENAME"
  else
    ./hack/catalog.sh > "$CATALOG_SOURCE_FILENAME"
  fi
  oc apply -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME" || return 1

  logger.success "CatalogSource installed successfully"
}

function tag_operator_image(){
  if [[ -n "${OPENSHIFT_BUILD_NAMESPACE:-}" ]]; then
    oc policy add-role-to-group system:image-puller "system:serviceaccounts:${OPERATORS_NAMESPACE}" --namespace="${OPENSHIFT_BUILD_NAMESPACE}" >/dev/null
    oc tag --insecure=false -n "${OPERATORS_NAMESPACE}" "${OPENSHIFT_REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:${OPERATOR} ${OPERATOR}:latest" >/dev/null
    echo "$INTERNAL_REGISTRY/$OPERATORS_NAMESPACE/$OPERATOR"
  fi
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource"
  oc delete --ignore-not-found=true -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME"
  rm -v "$CATALOG_SOURCE_FILENAME"
}
