#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/images.bash"

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

  local rootdir csv index_image

  default_serverless_operator_images

  index_image="${INDEX_IMAGE:-$SERVERLESS_INDEX}"

  # Build bundle and index images only when running in CI or when DOCKER_REPO_OVERRIDE is defined.
  # Otherwise the latest nightly build will be used for CatalogSource.
  if [ -n "$OPENSHIFT_CI" ] || [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    index_image=image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-index:latest
    rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

    csv="${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"

    logger.debug "Create a backup of the CSV so we don't pollute the repository."
    mkdir -p "${rootdir}/_output"
    cp "$csv" "${rootdir}/_output/bkp.yaml"

    if [ -n "$DOCKER_REPO_OVERRIDE" ]; then
      export SERVERLESS_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/serverless-knative-operator"
      export SERVERLESS_INGRESS="${DOCKER_REPO_OVERRIDE}/serverless-ingress"
      export SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/serverless-openshift-knative-operator"
    fi

    # Generate CSV from template to properly substitute operator images from env variables.
    "${rootdir}/hack/generate/csv.sh" templates/csv.yaml "$csv"

    if [[ "${OPENSHIFT_BUILD_NAME:-}" = serverless-source-image* ]]; then
      # Replace storage version migration images with quay.io variants for test purposes.
      override_storage_version_migration_images "$csv"
    fi

    # Replace registry.redhat.io references with Konflux quay.io for test purposes as
    # images in the former location are not published yet.
    sed -ri "s#(.*)${registry_redhat_io}/(.*@sha[0-9]+:[a-z0-9]+.*)#\1${registry}/\2#" "$csv"

    cat "$csv"

    build_image "serverless-bundle" "${rootdir}" "olm-catalog/serverless-operator/Dockerfile"

    logger.debug 'Undo potential changes to the CSV to not pollute the repository.'
    mv "${rootdir}/_output/bkp.yaml" "$csv"

    # TODO: Use proper secrets for OPM instead of unauthenticated user,
    # See https://github.com/operator-framework/operator-registry/issues/919

    # Allow OPM to pull the serverless-bundle from openshift-marketplace ns from internal registry.
    oc adm policy add-role-to-group system:image-puller system:unauthenticated --namespace openshift-marketplace

    # export ON_CLUSTER_BUILDS=true; make images
    # will push images to ${OLM_NAMESPACE} namespace, allow the ${OPERATORS_NAMESPACE} namespace to pull those images.
    oc adm policy add-role-to-group system:image-puller system:serviceaccounts:"${OPERATORS_NAMESPACE}" --namespace "${OLM_NAMESPACE}"

    local index_dorkerfile_path="olm-catalog/serverless-operator/index/Dockerfile"

    logger.debug "Create a backup of the index Dockerfile."
    cp "${index_dorkerfile_path}" "${rootdir}/_output/bkp.Dockerfile"

    # Replace bundle reference with previously built bundle
    bundle="${DEFAULT_SERVERLESS_BUNDLE%:*}" # Remove the tag from the match
    bundle="${DEFAULT_SERVERLESS_BUNDLE%@*}" # Remove the sha from the match
    if ! grep "${bundle}" "${rootdir}/${index_dorkerfile_path}"; then
      logger.error "Bundle ${bundle} not found in Dockerfile."
      return 1
    fi
    sed -ri "s#(.*)(${bundle})(:[a-z0-9]*)?(@sha[0-9]+:[a-z0-9]+)?(.*)#\1image-registry.openshift-image-registry.svc:5000/${OLM_NAMESPACE}/serverless-bundle:latest\5#" "${rootdir}/${index_dorkerfile_path}"

    build_image "serverless-index" "${rootdir}" "${index_dorkerfile_path}"

    logger.debug 'Undo potential changes to the index Dockerfile.'
    mv "${rootdir}/_output/bkp.Dockerfile" "${rootdir}/${index_dorkerfile_path}"
  else
    install_image_content_source_policy "$index_image"
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

function install_image_content_source_policy {
  local index tmpfile
  index="${1:?Pass index image as arg[1]}"
  logger.info "Install ImageContentSourcePolicy"
  tmpfile=$(mktemp /tmp/icsp.XXXXXX.yaml)
  cat > "$tmpfile" <<EOF
apiVersion: operator.openshift.io/v1alpha1
kind: ImageContentSourcePolicy
metadata:
  labels:
    operators.openshift.org/catalog: "true"
  name: serverless-image-content-source-policy
spec:
  repositoryDigestMirrors:
EOF

    oc adm catalog mirror "$index" "$registry" --manifests-only=true --to-manifests=iib-manifests/

    # The generated ICSP is incorrect as it replaces slashes in long repository paths with dashes and
    # includes third-party images. Create a proper ICSP based on the generated one.
    mirrors=$(yq read iib-manifests/imageContentSourcePolicy.yaml 'spec.repositoryDigestMirrors[*].source')
    while IFS= read -r line; do
      if  [[ $line == $registry_redhat_io || $line =~ $registry_redhat_io ]]; then
        img=${line##*/} # Get image name after last slash
        add_repository_digest_mirrors "$tmpfile" "${registry_redhat_io}/${img}" "${registry}/${img}"
      fi
    done <<< "$mirrors"

  [ -n "$OPENSHIFT_CI" ] && cat "$tmpfile"
  oc apply -f "$tmpfile"
}

function add_repository_digest_mirrors {
  echo "Add mirror image to '${1}' - $2 = $3"
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.repositoryDigestMirrors[+]
  value:
    mirrors: [ "${3}" ]
    source: "${2}"
EOF
}

# Dockerfiles might specify "FROM $XYZ" which fails OpenShift on-cluster
# builds. Replace the references with real images.
function replace_images() {
  local dockerfile_path tmp_dockerfile go_runtime go_builder
  dockerfile_path=${1:?Pass dockerfile path}
  tmp_dockerfile=$(mktemp /tmp/Dockerfile.XXXXXX)
  cp "${dockerfile_path}" "$tmp_dockerfile"

  if grep -q "GO_RUNTIME=" "$tmp_dockerfile"; then
    go_runtime=$(grep "GO_RUNTIME=" "$tmp_dockerfile" | cut -d"=" -f 2)
  fi

  if grep -q "GO_BUILDER=" "$tmp_dockerfile"; then
    go_builder=$(grep "GO_BUILDER=" "$tmp_dockerfile" | cut -d"=" -f 2)
  fi

  sed -e "s|\$GO_RUNTIME|${go_runtime:-}|" \
      -e "s|\$GO_BUILDER|${go_builder:-}|" -i "$tmp_dockerfile"

  echo "$tmp_dockerfile"
}

function build_image() {
  local name from_dir dockerfile_path tmp_dockerfile
  name=${1:?Pass a name of image to be built as arg[1]}
  from_dir=${2:?Pass context dir}
  dockerfile_path=${3:?Pass dockerfile path}
  tmp_dockerfile=$(replace_images "${from_dir}/${dockerfile_path}")

  logger.info "Using ${tmp_dockerfile} as Dockerfile"

  if ! oc get buildconfigs "$name" -n "$OLM_NAMESPACE" >/dev/null 2>&1; then
    logger.info "Create an image build for ${name}"
    oc -n "${OLM_NAMESPACE}" new-build \
      --strategy=docker --name "$name" --dockerfile "$(cat "${tmp_dockerfile}")"
  else
    logger.info "${name} image build is already created"
  fi

  logger.info 'Build the image in the cluster-internal registry.'
  oc -n "${OLM_NAMESPACE}" start-build "${name}" --from-dir "${from_dir}" -F
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource $OPERATOR"
  oc delete imagecontentsourcepolicy --ignore-not-found=true serverless-image-content-source-policy
  oc delete catalogsource --ignore-not-found=true -n "$OLM_NAMESPACE" "$OPERATOR"
  oc delete service --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete deployment --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete configmap --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index-sha1sums
  oc delete buildconfig --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete configmap --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-bundle-sha1sums
  oc delete buildconfig --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-bundle
  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 300 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]"
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

# Use images from quay registry in order to test issues such as SRVCOM-1873
# during upgrades. The issue is only reproducible if Knative version is identical
# before/after upgrade but the migrator Job spec/image changes. Before upgrade,
# the images are pulled from CI registry and after upgrade they're pulled from
# quay.io. This way the Job spec is changes even though Knative version
# remains same.
function override_storage_version_migration_images {
  local csv images name version
  csv=${1:?Pass csv as arg[1]}
  # Get all storage version migration images.
  while IFS=$'\n' read -r line; do
    images+=("$line");
  done < <(grep storage-version-migration "$csv" | grep "image:" | awk '{ print $2 }' | awk -F"\"" '{ print $2 }')
  for image_pullspec in "${images[@]}"; do
    name=$(echo "$image_pullspec" | awk -F":" '{print $1}' | awk -F"/" '{print $NF}')
    version=$(echo "$image_pullspec" | awk -F":" '{ print $2 }')
    sed -i "s,${image_pullspec},quay.io/openshift-knative/${name}:${version}," "$csv"
  done
}
