#!/usr/bin/env bash

# Setup image pull permissions for OLMv0 namespaces
function setup_olmv0_image_pull_permissions {
  logger.info "Setting up image pull permissions for OLMv0 namespaces"

  # TODO: Use proper secrets for OPM instead of unauthenticated user,
  # See https://github.com/operator-framework/operator-registry/issues/919

  # Allow OPM to pull the serverless-bundle from openshift-serverless-builds ns from internal registry.
  oc adm policy add-role-to-group system:image-puller system:unauthenticated --namespace "${OLM_NAMESPACE}"
  oc adm policy add-role-to-group system:image-puller system:unauthenticated --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"

  # export ON_CLUSTER_BUILDS=true; make images
  # will push images to ${ON_CLUSTER_BUILDS_NAMESPACE} namespace, allow the ${OPERATORS_NAMESPACE} namespace to pull those images.
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${ON_CLUSTER_BUILDS_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${OLM_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
  oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${OPERATORS_NAMESPACE}" --namespace "${ON_CLUSTER_BUILDS_NAMESPACE}"
}

function remove_installplan {
  local install_plan csv
  csv="${1:?Pass a CSV as arg[1]}"
  logger.info "Removing installplan for $csv"
  install_plan=$(find_install_plan "$csv")
  if [[ -n $install_plan ]]; then
    oc delete "$install_plan" -n "${OPERATORS_NAMESPACE}"
  else
    logger.debug "No install plan for $csv"
  fi
}

function approve_csv {
  local csv_version install_plan channel
  csv_version=${1:?Pass a CSV as arg[1]}
  channel=${2:?Pass channel as arg[2]}

  logger.info 'Ensure channel and source is set properly'
  oc patch subscriptions.operators.coreos.com "$OPERATOR" -n "${OPERATORS_NAMESPACE}" \
    --type 'merge' \
    --patch '{"spec": {"channel": "'"${channel}"'", "source": "'"${OLM_SOURCE}"'"}}'

  logger.info 'Wait for the installplan to be available'
  timeout 900 "[[ -z \$(find_install_plan ${csv_version}) ]]"

  install_plan=$(find_install_plan "${csv_version}")
  oc patch "$install_plan" -n "${OPERATORS_NAMESPACE}" \
    --type merge --patch '{"spec":{"approved":true}}'

  if ! timeout 300 "[[ \$(oc get ClusterServiceVersion $csv_version -n ${OPERATORS_NAMESPACE} -o jsonpath='{.status.phase}') != Succeeded ]]" ; then
    oc get ClusterServiceVersion "$csv_version" -n "${OPERATORS_NAMESPACE}" -o yaml || true
    return 105
  fi
}

function find_install_plan {
  local csv="${1:-Pass a CSV as arg[1]}"
  for plan in $(oc get installplan -n "${OPERATORS_NAMESPACE}" --no-headers -o name); do
    if [[ $(oc get "$plan" -n "${OPERATORS_NAMESPACE}" -o=jsonpath='{.spec.clusterServiceVersionNames}' | grep -c "$csv") -eq 1 && \
      $(oc get "$plan" -n "${OPERATORS_NAMESPACE}" -o=jsonpath="{.status.bundleLookups[0].catalogSourceRef.name}" | grep -c "$OLM_SOURCE") -eq 1 ]]
    then
      echo "$plan"
      return 0
    fi
  done
  echo ""
}

function ensure_serverless_installed_olmv0 {
  logger.info 'Check if Serverless is installed'
  if check_serverless_already_installed; then
    logger.success 'Serverless is already installed.'
    return 0
  fi

  # Deploy config-logging configmap before running serving-operator pod.
  # Otherwise, we cannot change log level by configmap.
  enable_debug_log

  local csv
  determine_csv_version

  # Remove installplan from previous installations, leaving this would make the operator
  # upgrade to the latest version immediately
  if [[ "$csv" != "$CURRENT_CSV" ]]; then
    remove_installplan "$CURRENT_CSV"
  fi

  if [[ ${SKIP_OPERATOR_SUBSCRIPTION:-} != "true" ]]; then
    logger.info "Installing Serverless version $csv"
    deploy_serverless_operator_olmv0 "$csv"
  fi

  install_knative_resources "${csv#serverless-operator.v}"

  logger.success "Serverless is installed: $csv"
}

function deploy_serverless_operator_olmv0 {
  local csv tmpfile
  csv="${1:?Pass a CSV as arg[1]}"
  logger.info "Install the Serverless Operator: ${csv}"
  tmpfile=$(mktemp /tmp/subscription.XXXXXX.yaml)
  cat > "$tmpfile" <<EOF
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: "${OPERATOR}"
  namespace: "${OPERATORS_NAMESPACE}"
spec:
  channel: "${OLM_CHANNEL}"
  name: "${OPERATOR}"
  source: "${OLM_SOURCE}"
  sourceNamespace: "${OLM_NAMESPACE}"
  installPlanApproval: Manual
  startingCSV: "${csv}"
EOF
  [ -n "$OPENSHIFT_CI" ] && cat "$tmpfile"
  oc apply -f "$tmpfile"

  # Approve the initial installplan automatically
  approve_csv "$csv" "$OLM_CHANNEL"
}

function teardown_serverless_olmv0 {
  logger.warn '😭  Teardown Serverless...'

  teardown_knative_resources

  logger.info 'Deleting subscription'
  oc delete subscriptions.operators.coreos.com \
    -n "${OPERATORS_NAMESPACE}" "${OPERATOR}" \
    --ignore-not-found
  logger.info 'Deleting ClusterServiceVersion'
  for csv in $(set +o pipefail && oc get csv -n "${OPERATORS_NAMESPACE}" --no-headers 2>/dev/null \
      | grep "${OPERATOR}" | cut -f1 -d' '); do
    oc delete csv -n "${OPERATORS_NAMESPACE}" "${csv}"
  done

  teardown_operator_and_crds

  logger.success 'Serverless has been uninstalled.'
}

# ============================================================================
# CatalogSource Management (OLMv0)
# ============================================================================

function ensure_catalogsource_installed {
  logger.info 'Check if CatalogSource is installed'
  if oc get catalogsource "$OPERATOR" -n "$OLM_NAMESPACE" > /dev/null 2>&1; then
    logger.success 'CatalogSource is already installed.'
    return 0
  fi
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"

  ensure_catalog_pods_running

  local index_image

  default_serverless_operator_images

  # env variable INDEX_IMAGE is exported by default_serverless_operator_images
  index_image="${INDEX_IMAGE}"

  # Build bundle and index images only when running in CI or when DOCKER_REPO_OVERRIDE is defined,
  # unless overridden by FORCE_KONFLUX_INDEX.
  if { [ -n "$OPENSHIFT_CI" ] || [ -n "$DOCKER_REPO_OVERRIDE" ]; } && [ -z "${FORCE_KONFLUX_INDEX:-}" ]; then
    setup_olmv0_image_pull_permissions
    build_bundle_and_index_images
    # index_image is set by build_bundle_and_index_images
  else
    apply_icsp_for_konflux_index "$index_image"
  fi

  logger.info 'Install the catalogsource.'
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${OPERATOR}
spec:
  image: ${index_image}
  displayName: "Serverless Operator"
  publisher: Red Hat
  sourceType: grpc
EOF

  # Ensure the Index pod is created with the right pull secret. The Pod's service account needs
  # to be linked with the right pull secret before creating the Pod. This is to prevent race conditions.
  timeout 120 "[[ \$(oc -n $OLM_NAMESPACE get pods -l olm.catalogSource=serverless-operator --no-headers | wc -l) != 1 ]]"
  index_pod=$(oc -n "$OLM_NAMESPACE" get pods -l olm.catalogSource=serverless-operator -oname)
  if ! oc -n "$OLM_NAMESPACE" get "$index_pod" -ojsonpath='{.spec.imagePullSecrets}' | grep dockercfg &>/dev/null; then
    timeout 120 "[[ \$(oc -n $OLM_NAMESPACE get sa serverless-operator -ojsonpath='{.imagePullSecrets}' | grep -c dockercfg) == 0 ]]"
    oc -n "$OLM_NAMESPACE" delete pods -l olm.catalogSource=serverless-operator
  fi

  logger.success "CatalogSource installed successfully"
}

function delete_catalogsource {
  logger.info "Deleting CatalogSource $OPERATOR"
  oc delete catalogsource --ignore-not-found=true -n "$OLM_NAMESPACE" "$OPERATOR"
  oc delete service --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete deployment --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete configmap --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index-sha1sums
  oc delete buildconfig --ignore-not-found=true -n "$ON_CLUSTER_BUILDS_NAMESPACE" serverless-index
  oc delete configmap --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-bundle-sha1sums
  oc delete buildconfig --ignore-not-found=true -n "$ON_CLUSTER_BUILDS_NAMESPACE" serverless-bundle
  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 300 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]"
  oc delete imagecontentsourcepolicy --ignore-not-found=true serverless-image-content-source-policy
  logger.success 'CatalogSource deleted'
}

function add_user {
  local name pass occmd rootdir
  name=${1:?Pass a username as arg[1]}
  pass=${2:?Pass a password as arg[2]}

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  logger.info "Creating user $name:***"
  if oc get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.debug 'Secret htpass-secret already existed, updating it.'
    # Add a newline to the end of the file if not already present (htpasswd will butcher it otherwise).
    [ -n "$(tail -c1 users.htpasswd)" ] && echo >> users.htpasswd
  else
    touch users.htpasswd
  fi

  htpasswd -b users.htpasswd "$name" "$pass"

  oc create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | oc apply -f -

  if oc get oauth.config.openshift.io cluster &>/dev/null; then
    oc replace -f "${rootdir}/openshift/identity/htpasswd.yaml"
  else
    oc apply -f "${rootdir}/openshift/identity/htpasswd.yaml"
  fi

  logger.debug 'Generate kubeconfig'

  if oc config current-context >&/dev/null; then
    ctx=$(oc config current-context)
    cluster=$(oc config view -ojsonpath="{.contexts[?(@.name == \"$ctx\")].context.cluster}")
    server=$(oc config view -ojsonpath="{.clusters[?(@.name == \"$cluster\")].cluster.server}")
    logger.debug "Context: $ctx, Cluster: $cluster, Server: $server"
  else
    # Fallback to in-cluster api server service.
    server="https://kubernetes.default.svc"
  fi

  occmd="bash -c '! oc login --kubeconfig=${name}.kubeconfig --insecure-skip-tls-verify=true --username=${name} --password=${pass} ${server} > /dev/null'"
  timeout 600 "${occmd}"

  logger.info "Kubeconfig for user ${name} created"
}
