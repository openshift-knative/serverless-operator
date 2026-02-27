#!/usr/bin/env bash

# Shared catalog utilities used by both OLMv0 and OLMv1

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/images.bash"

# Build bundle and index images in CI environment
# Sets index_image and bundle_image variables in the calling scope
function build_bundle_and_index_images {
  local rootdir csv
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  index_image=image-registry.openshift-image-registry.svc:5000/$ON_CLUSTER_BUILDS_NAMESPACE/serverless-index:latest
  bundle_image=image-registry.openshift-image-registry.svc:5000/$ON_CLUSTER_BUILDS_NAMESPACE/serverless-bundle:latest

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
  #
  # Pass "true" to replace registry.redhat.io references with Konflux quay.io for test purposes as
  # images in the former location are not published yet.
  "${rootdir}/hack/generate/csv.sh" templates/csv.yaml "$csv" "true"

  cat "$csv"

  build_image "serverless-bundle" "${rootdir}" "olm-catalog/serverless-operator/Dockerfile"

  logger.debug 'Undo potential changes to the CSV to not pollute the repository.'
  mv "${rootdir}/_output/bkp.yaml" "$csv"

  local index_dockerfile_path="olm-catalog/serverless-operator-index/Dockerfile"

  logger.debug "Create a backup of the index Dockerfile."
  cp "${index_dockerfile_path}" "${rootdir}/_output/bkp.Dockerfile"

  # Replace bundle reference with previously built bundle
  export SERVERLESS_BUNDLE="${bundle_image}"
  "${rootdir}/hack/generate/dockerfile.sh" "${rootdir}/templates/index.Dockerfile" "${index_dockerfile_path}"

  build_image "serverless-index" "${rootdir}" "${index_dockerfile_path}"

  logger.debug 'Undo potential changes to the index Dockerfile.'
  mv "${rootdir}/_output/bkp.Dockerfile" "${rootdir}/${index_dockerfile_path}"
}

# Apply ICSP for Konflux index and wait for machineconfigpool
function apply_icsp_for_konflux_index {
  local index_image="${1:?Pass index image as arg[1]}"

  local tmpfile idms_tmpfile
  tmpfile=$(mktemp /tmp/icsp.XXXXXX.yaml)
  idms_tmpfile=$(mktemp /tmp/idms.XXXXXX.yaml)
  # Use ImageContentSourcePolicy only with the FBC from Konflux as
  # updating machine config pools takes a while.
  # shellcheck disable=SC2154
  create_image_content_source_policy "$index_image" "$registry_redhat_io" "$registry_quay" "$registry_quay_previous" "$registry_quay_next" "$tmpfile" "$idms_tmpfile"
  [ -n "$OPENSHIFT_CI" ] && cat "$tmpfile"
  if oc apply -f "$tmpfile"; then
    echo "Wait for machineconfigpool update to start"
    timeout 120 "[[ True != \$(oc get machineconfigpool --no-headers=true '-o=custom-columns=UPDATING:.status.conditions[?(@.type==\"Updating\")].status' | uniq) ]]"
    echo "Wait until all machineconfigpools are updated"
    timeout 1800 "[[ True != \$(oc get machineconfigpool --no-headers=true '-o=custom-columns=UPDATED:.status.conditions[?(@.type==\"Updated\")].status' | uniq) ]]"
  fi
}

# Create ImageContentSourcePolicy and ImageDigestMirrorSet
function create_image_content_source_policy {
  local index registry_source registry_target rootdir
  index="${1:?Pass index image as arg[1]}"
  registry_source="${2:?Pass source registry arg[2]}"
  registry_target="${3:?Pass target registry arg[3]}"
  registry_target_previous="${4:?Pass previous target registry arg[4]}"
  registry_target_next="${5:?Pass next target registry arg[5]}"
  image_content_source_policy_output_file="${6:?Pass output file arg[6]}"
  image_digest_mirror_output_file="${7:?Pass image_digest_mirror_output_file arg[7]}"

  logger.info "Install ImageContentSourcePolicy"
  cat > "$image_content_source_policy_output_file" <<EOF
apiVersion: operator.openshift.io/v1alpha1
kind: ImageContentSourcePolicy
metadata:
  labels:
    operators.openshift.org/catalog: "true"
  name: serverless-image-content-source-policy
spec:
  repositoryDigestMirrors:
EOF

  cat > "$image_digest_mirror_output_file" <<EOF
apiVersion: operator.openshift.io/v1alpha1
kind: ImageDigestMirrorSet
metadata:
  name: mirror-set
spec:
  imageDigestMirrors:
EOF

  rm -rf iib-manifests
  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  if oc adm catalog mirror "$index" "$registry_target" --manifests-only=true --to-manifests=iib-manifests/ ; then
    mirrors=$(yq read iib-manifests/imageContentSourcePolicy.yaml 'spec.repositoryDigestMirrors[*].source' | sort)
  else
    mirrors=$(yq read "${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml" 'spec.relatedImages[*].image' | grep "${registry_source}" | sort | uniq)
  fi
  # The generated ICSP is incorrect as it replaces slashes in long repository paths with dashes and
  # includes third-party images. Create a proper ICSP based on the generated one.
  while IFS= read -r line; do
    # shellcheck disable=SC2053
    if  [[ $line == $registry_source || $line =~ $registry_source ]]; then
      img=${line##*/} # Get image name after last slash
      img=${img%@*} # Remove sha
      img=${img%:*} # Remove tag

      if [[ "${img}" =~ ^serverless-openshift-kn-rhel[0-9]+-operator$ ]]; then
        # serverless-openshift-kn-operator is special, as it has rhel in the middle of the name
        # see https://redhat-internal.slack.com/archives/CKR568L8G/p1729684088850349
        target_img="serverless-openshift-kn-operator"
      elif [[ "${img}" == "serverless-operator-bundle" ]]; then
        # serverless-operator-bundle is special, as it is named only serverless-bundle in quay
        target_img="serverless-bundle"
      else
        # for other images simply remove the -rhelXYZ suffix
        target_img=${img%-rhel*}
      fi

      echo "Processing line: ${line}, image ${img} -> target image: ${target_img}"

      local mirror1="${registry_target}/${target_img}"
      local mirror2="${registry_target_previous}/${target_img}"
      local mirror3="${registry_target_next}/${target_img}"

      add_repository_digest_mirrors "$image_content_source_policy_output_file" "${registry_source}/${img}" "${mirror1}" "${mirror2}" "${mirror3}"
      add_image_digest_mirrors "$image_digest_mirror_output_file" "${registry_source}/${img}" "${mirror1}" "${mirror2}" "${mirror3}"
    fi
  done <<< "$mirrors"
}

function add_repository_digest_mirrors {
  echo "Add mirror image to '${1}' - $2 = $3, $4, $5"
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.repositoryDigestMirrors[+]
  value:
    mirrors: [ "${3}", "${4}", "${5}" ]
    source: "${2}"
EOF
}

function add_image_digest_mirrors {
  echo "Add mirror image to '${1}' - $2 = $3, $4, $5"
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.imageDigestMirrors[+]
  value:
    mirrors: [ "${3}", "${4}", "${5}" ]
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

  if grep -q "OPM_IMAGE=" "$tmp_dockerfile"; then
      opm_image=$(grep "OPM_IMAGE=" "$tmp_dockerfile" | cut -d"=" -f 2)
    fi

  sed -e "s|\$GO_RUNTIME|${go_runtime:-}|" \
      -e "s|\$GO_BUILDER|${go_builder:-}|" \
      -e "s|\$OPM_IMAGE|${opm_image:-}|" -i "$tmp_dockerfile"

  echo "$tmp_dockerfile"
}

function build_image() {
  local name from_dir dockerfile_path tmp_dockerfile image_stream_tag from_kind
  name=${1:?Pass a name of image to be built as arg[1]}
  from_dir=${2:?Pass context dir}
  dockerfile_path=${3:?Pass dockerfile path}
  tmp_dockerfile=$(replace_images "${from_dir}/${dockerfile_path}")

  logger.info "Using ${tmp_dockerfile} as Dockerfile"

  if ! oc get buildconfigs "$name" -n "$ON_CLUSTER_BUILDS_NAMESPACE" >/dev/null 2>&1; then
    logger.info "Create an image build for ${name}"
    oc -n "${ON_CLUSTER_BUILDS_NAMESPACE}" new-build \
      --strategy=docker --name "$name" --dockerfile "$(cat "${tmp_dockerfile}")"

    from_kind=$(oc get BuildConfig -n "${ON_CLUSTER_BUILDS_NAMESPACE}" "$name" -o json | \
      jq -r '.spec.strategy.dockerStrategy.from.kind')
    if [ "ImageStreamTag" = "$from_kind" ]; then
      image_stream_tag=$(oc get BuildConfig -n "${ON_CLUSTER_BUILDS_NAMESPACE}" "$name" -o json | \
        jq -r '.spec.strategy.dockerStrategy.from.name')

      logger.info "Wait for the ${image_stream_tag} ImageStreamTag to be imported"
      timeout 60 "! oc get imagestreamtag -n \"${ON_CLUSTER_BUILDS_NAMESPACE}\" \"$image_stream_tag\" -o json | jq -re .image.dockerImageReference"
    fi
  else
    logger.info "${name} image build is already created"
  fi

  logger.info 'Build the image in the cluster-internal registry.'
  oc -n "${ON_CLUSTER_BUILDS_NAMESPACE}" start-build "${name}" --from-dir "${from_dir}" -F
}
