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

  # Use current build images in CI.
  if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then
    export IMAGE_KNATIVE_SERVING_OPERATOR="registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:knative-serving-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/${OPENSHIFT_BUILD_NAMESPACE}/stable:knative-openshift-ingress"
  # Use DOCKER_REPO_OVERRIDE for developing locally
  elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    export IMAGE_KNATIVE_SERVING_OPERATOR="${DOCKER_REPO_OVERRIDE}/knative-serving-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress"
  # Use CI_PROMOTION_NAME to use current promoted images from CI
  elif [ -n "$CI_PROMOTION_NAME" ]; then
    export IMAGE_KNATIVE_SERVING_OPERATOR="registry.svc.ci.openshift.org/openshift/${CI_PROMOTION_NAME}:knative-serving-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/${CI_PROMOTION_NAME}:knative-openshift-ingress"
  fi

  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  ${rootdir}/hack/catalog.sh | envsubst | oc apply -n "$OPERATORS_NAMESPACE" -f - || return 1

  logger.success "CatalogSource installed successfully"
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
