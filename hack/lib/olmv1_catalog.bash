#!/usr/bin/env bash

function ensure_clustercatalog_installed {
  logger.info 'Check if ClusterCatalog is installed'
  if oc get clustercatalog "$OLMV1_CATALOG_NAME" > /dev/null 2>&1; then
    logger.success 'ClusterCatalog is already installed.'
    return 0
  fi
  install_clustercatalog
}

function install_clustercatalog {
  logger.info "Installing ClusterCatalog"

  local index_image

  default_serverless_operator_images

  # env variable INDEX_IMAGE is exported by default_serverless_operator_images
  index_image="${INDEX_IMAGE}"

  # Build bundle and index images only when running in CI or when DOCKER_REPO_OVERRIDE is defined,
  # unless overridden by FORCE_KONFLUX_INDEX.
  if { [ -n "$OPENSHIFT_CI" ] || [ -n "$DOCKER_REPO_OVERRIDE" ]; } && [ -z "${FORCE_KONFLUX_INDEX:-}" ]; then
    # Set up image pull permissions for OLMv1 namespaces
    setup_olmv1_image_pull_permissions
    # Use shared image building function from catalogsource.bash
    build_bundle_and_index_images
    # index_image is set by build_bundle_and_index_images
  else
    # Use shared ICSP function from catalogsource.bash
    apply_icsp_for_konflux_index "$index_image"
  fi

  logger.info 'Install the ClusterCatalog.'
  cat <<EOF | oc apply -f -
apiVersion: olm.operatorframework.io/v1
kind: ClusterCatalog
metadata:
  name: ${OLMV1_CATALOG_NAME}
spec:
  priority: ${OLMV1_CATALOG_PRIORITY}
  source:
    type: Image
    image:
      ref: ${index_image}
      pollInterval: 10m
EOF

  logger.info 'Wait for ClusterCatalog to start serving'
  oc wait --for=condition=Serving clustercatalog/${OLMV1_CATALOG_NAME} --timeout=120s

  logger.success "ClusterCatalog installed successfully"
}

function setup_olmv1_image_pull_permissions {
  logger.info "Setting up image pull permissions for OLMv1 namespaces"

  # TODO: Use proper secrets for OPM instead of unauthenticated user,
  # See https://github.com/operator-framework/operator-registry/issues/919

  # Allow OPM to pull the serverless-bundle from on-cluster builds namespace from internal registry.
  oc adm policy add-role-to-group system:image-puller system:unauthenticated --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"

  # Allow build pods in the on-cluster builds namespace to pull images from the same namespace
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${ON_CLUSTER_BUILDS_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"

  # Allow catalogd and operator-controller to pull from on-cluster builds namespace
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${CATALOGD_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${OPERATOR_CONTROLLER_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${OPERATORS_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
}

function delete_clustercatalog {
  logger.info "Deleting ClusterCatalog $OLMV1_CATALOG_NAME"
  oc delete clustercatalog --ignore-not-found=true "$OLMV1_CATALOG_NAME"
  oc delete buildconfig --ignore-not-found=true -n "$ON_CLUSTER_BUILDS_NAMESPACE" serverless-index
  oc delete buildconfig --ignore-not-found=true -n "$ON_CLUSTER_BUILDS_NAMESPACE" serverless-bundle
  oc delete imagecontentsourcepolicy --ignore-not-found=true serverless-image-content-source-policy
  logger.success 'ClusterCatalog deleted'
}

# ============================================================================
# Serverless Operator Deployment (OLMv1)
# ============================================================================

function setup_olmv1_serviceaccount {
  logger.info "Setting up ServiceAccount for OLMv1 installation"

  # Create ServiceAccount
  cat <<EOF | oc apply -n "${OPERATORS_NAMESPACE}" -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ${OLMV1_INSTALLER_SA}
  namespace: ${OPERATORS_NAMESPACE}
EOF

  # Create ClusterRoleBinding with cluster-admin
  cat <<EOF | oc apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: ${OLMV1_INSTALLER_SA}-cluster-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: ${OLMV1_INSTALLER_SA}
  namespace: ${OPERATORS_NAMESPACE}
EOF

  logger.success "ServiceAccount ${OLMV1_INSTALLER_SA} created with cluster-admin role"
}

function deploy_serverless_operator_olmv1 {
  local csv version
  csv="${1:?Pass a CSV as arg[1]}"

  # Extract version from CSV name: serverless-operator.v1.37.0 -> 1.37.0
  version="${csv#serverless-operator.v}"

  logger.info "Deploy Serverless Operator via ClusterExtension: ${csv} (version: ${version})"

  # Setup ServiceAccount if it doesn't exist
  if ! oc get serviceaccount "${OLMV1_INSTALLER_SA}" -n "${OPERATORS_NAMESPACE}" >/dev/null 2>&1; then
    setup_olmv1_serviceaccount
  fi

  # Create or update ClusterExtension
  cat <<EOF | oc apply -f -
apiVersion: olm.operatorframework.io/v1
kind: ClusterExtension
metadata:
  name: ${OLMV1_CLUSTEREXTENSION_NAME}
spec:
  namespace: ${OPERATORS_NAMESPACE}
  serviceAccount:
    name: ${OLMV1_INSTALLER_SA}
  source:
    sourceType: Catalog
    catalog:
      packageName: serverless-operator
      version: ${version}
      upgradeConstraintPolicy: ${OLMV1_UPGRADE_CONSTRAINT_POLICY}
EOF

  # Wait for ClusterExtension to be ready
  wait_for_clusterextension_ready "$csv" "$version"
}

function wait_for_clusterextension_ready {
  local csv version
  csv="${1:?Pass a CSV as arg[1]}"
  version="${2:?Pass a version as arg[2]}"

  logger.info "Wait for ClusterExtension to be ready"

  # Wait for Progressing condition with Succeeded reason
  if ! timeout 600 "[[ \$(oc get clusterextension ${OLMV1_CLUSTEREXTENSION_NAME} -o jsonpath='{.status.conditions[?(@.type==\"Progressing\")].reason}') != Succeeded ]]" ; then
    oc get clusterextension "${OLMV1_CLUSTEREXTENSION_NAME}" -o yaml || true
    logger.error "ClusterExtension failed to become ready"
    return 105
  fi

  # Verify installed bundle version matches requested version
  local installed_version
  installed_version=$(oc get clusterextension "${OLMV1_CLUSTEREXTENSION_NAME}" -o jsonpath='{.status.install.bundle.version}')

  if [[ "$installed_version" != "$version" ]]; then
    logger.error "Installed version ($installed_version) does not match requested version ($version)"
    return 106
  fi

  logger.success "ClusterExtension is ready with version ${version}"
}

function ensure_serverless_installed_olmv1 {
  logger.info 'Check if Serverless is installed (OLMv1)'
  if check_serverless_already_installed; then
    logger.success 'Serverless is already installed.'
    return 0
  fi

  # Deploy config-logging configmap before running serving-operator pod.
  # Otherwise, we cannot change log level by configmap.
  enable_debug_log

  local csv
  determine_csv_version

  if [[ ${SKIP_OPERATOR_SUBSCRIPTION:-} != "true" ]]; then
    logger.info "Installing Serverless version $csv (OLMv1)"
    deploy_serverless_operator_olmv1 "$csv"
  fi

  install_knative_resources "${csv#serverless-operator.v}"

  logger.success "Serverless is installed: $csv (OLMv1)"
}

# ============================================================================
# Serverless Teardown (OLMv1)
# ============================================================================

function teardown_serverless_olmv1 {
  logger.warn '😭  Teardown Serverless (OLMv1)...'

  # Use shared helper for Knative resources teardown
  teardown_knative_resources

  logger.info 'Deleting ClusterExtension'
  oc delete clusterextension \
    "${OLMV1_CLUSTEREXTENSION_NAME}" \
    --ignore-not-found

  # Use shared helper for operator and CRDs teardown
  teardown_operator_and_crds

  logger.success 'Serverless has been uninstalled (OLMv1).'
}

# ============================================================================
# Upgrade/Downgrade Operations (OLMv1)
# ============================================================================

function upgrade_serverless_olmv1 {
  local csv version
  csv="${1:?Pass a CSV as arg[1]}"
  version="${csv#serverless-operator.v}"

  logger.info "Upgrading Serverless to ${csv} (version: ${version}) via OLMv1"

  # Patch ClusterExtension with new version (policy stays CatalogProvided)
  oc patch clusterextension "${OLMV1_CLUSTEREXTENSION_NAME}" \
    --type merge \
    --patch "{\"spec\":{\"source\":{\"catalog\":{\"version\":\"${version}\"}}}}"

  # Wait for upgrade to complete
  wait_for_clusterextension_ready "$csv" "$version"

  logger.success "Serverless upgraded to ${csv}"
}

function downgrade_serverless_olmv1 {
  local csv version
  csv="${1:?Pass a CSV as arg[1]}"
  version="${csv#serverless-operator.v}"

  logger.info "Downgrading Serverless to ${csv} (version: ${version}) via OLMv1"

  # Patch ClusterExtension with new version + change policy to SelfCertified
  oc patch clusterextension "${OLMV1_CLUSTEREXTENSION_NAME}" \
    --type merge \
    --patch "{\"spec\":{\"source\":{\"catalog\":{\"version\":\"${version}\",\"upgradeConstraintPolicy\":\"SelfCertified\"}}}}"

  # Wait for downgrade to complete
  wait_for_clusterextension_ready "$csv" "$version"

  logger.success "Serverless downgraded to ${csv}"
}
