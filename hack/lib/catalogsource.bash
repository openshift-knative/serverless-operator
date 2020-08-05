#!/usr/bin/env bash

function ensure_catalogsource_installed {
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"
  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # HACK: Allow to run the image as root
  oc adm policy add-scc-to-user anyuid -z default -n openshift-marketplace

  csv="${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"
  mv "$csv" "${rootdir}/raw.yaml"

  # Determine if we're running locally or in CI.
  if [ -n "$OPENSHIFT_CI" ]; then
    export IMAGE_KNATIVE_OPERATOR="${IMAGE_FORMAT//\$\{component\}/knative-operator}"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${IMAGE_FORMAT//\$\{component\}/knative-openshift-ingress}"
  elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    export IMAGE_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/knative-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress"
  else
    export IMAGE_KNATIVE_OPERATOR="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:knative-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:knative-openshift-ingress"
  fi

  rm "$csv"
  cat "${rootdir}/raw.yaml" | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' > "$csv"

  cat "$csv"

  oc -n openshift-marketplace new-build --binary --strategy=docker --name serverless-bundle
  oc -n openshift-marketplace start-build serverless-bundle --from-dir olm-catalog/serverless-operator -F

  mv "${rootdir}/raw.yaml" "$csv"

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
