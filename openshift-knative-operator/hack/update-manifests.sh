#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds.yaml eventing-core.yaml in-memory-channel.yaml mt-channel-broker.yaml eventing-post-install.yaml)

# This excludes the gateways and peerauthentication settings as we want customers to do
# manipulate those.
istio_files=(networkpolicy-mesh 200-clusterrole 400-config-istio 500-controller 500-webhook-deployment 500-webhook-secret 500-webhook-service 600-mutating-webhook 600-validating-webhook)

kourier_files=(kourier config-network)

export KNATIVE_EVENTING_MANIFESTS_DIR=${KNATIVE_EVENTING_MANIFESTS_DIR:-""}
export KNATIVE_SERVING_MANIFESTS_DIR=${KNATIVE_SERVING_MANIFESTS_DIR:-""}
export KNATIVE_SERVING_TEST_MANIFESTS_DIR=${KNATIVE_SERVING_TEST_MANIFESTS_DIR:-""}

function download_serving {
  component=$1
  version=$2
  shift
  shift

  files=("$@")
  echo "Files: ${files[*]}"

  component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
  target_dir="${component_dir}/${version:1}"
  rm -r "$component_dir"
  mkdir -p "$target_dir"

  branch=$(metadata.get dependencies.serving_artifacts_branch)
  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"

    if [[ ${KNATIVE_SERVING_MANIFESTS_DIR} = "" ]]; then
      url="https://raw.githubusercontent.com/openshift/knative-serving/${branch}/openshift/release/artifacts/$index-$file"
      wget --no-check-certificate "$url" -O "$target_file"
    else
      cp "${KNATIVE_SERVING_MANIFESTS_DIR}/${file}" "$target_file"
    fi
    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

function download_eventing {
  component=$1
  version=$2
  shift
  shift

  files=("$@")
  echo "Files: ${files[*]}"

  component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
  target_dir="${component_dir}/${version/knative-v/}" # remove `knative-v` prefix
  rm -r "$component_dir"
  mkdir -p "$target_dir"

  branch=$(metadata.get dependencies.eventing_artifacts_branch)
  for ((i = 0; i < ${#files[@]}; i++)); do
    index=$(( i+1 ))
    file="${files[$i]}"
    target_file="$target_dir/$index-$file"
    if [[ ${KNATIVE_EVENTING_MANIFESTS_DIR} = "" ]]; then

      url="https://raw.githubusercontent.com/openshift/knative-eventing/${branch}/openshift/release/artifacts/$file"
      echo "Downloading file from ${url}"
      wget --no-check-certificate "$url" -O "$target_file"
    else
      cp "${KNATIVE_EVENTING_MANIFESTS_DIR}/${file}" "$target_file"
    fi

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

function download_ingress {
  component=$1
  version=$2
  shift
  shift

  files=("$@")
  echo "Files: ${files[*]}"

  ingress_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
  SKIP_TARGET_DELETION=${SKIP_TARGET_DELETION:-false}
  if [[ "${SKIP_TARGET_DELETION}" == "false" ]]; then
    rm -rf "$ingress_dir"
  fi
  mkdir -p "$ingress_dir"

  branch=$(metadata.get "dependencies.${component/-/_}_artifacts_branch")
  index=0
  for (( i=0; i<${#files[@]}; i++ ));
  do

    file="${files[$i]}.yaml"
    ingress_target_file="$ingress_dir/$index-$file"
    url="https://raw.githubusercontent.com/openshift-knative/${component}/${branch}/openshift/release/artifacts/$index-$file"

    wget --no-check-certificate "$url" -O "$ingress_target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$ingress_target_file"
    index=$(( i+1 ))
  done
}

#
# DOWNLOAD SERVING
#

# When openshift/knative-serving uses this repo to run a job (eg. PR against openshift/knative-serving) it will use a minimum
# setup with net-kourier. Thus it will not use the release artifacts generated under openshift-knative-operator/cmd/kodata/knative-serving.
# Instead openshift/knative-serving uses its own generated ci manifests and sets KNATIVE_SERVING_TEST_MANIFESTS_DIR.
# Extensive Serving testing is done at this repo only. For the latter we do use manifests under openshift-knative-operator/cmd/kodata/knative-serving which are fetched from the midstream
# repo.
if [[ ${KNATIVE_SERVING_TEST_MANIFESTS_DIR} = "" ]]; then
  download_serving serving "${KNATIVE_SERVING_VERSION}" "${serving_files[@]}"
fi

#
# DOWNLOAD INGRESS
#

download_ingress net-istio "v$(metadata.get dependencies.net_istio)" "${istio_files[@]}"

SKIP_TARGET_DELETION=true download_ingress net-kourier "v$(metadata.get dependencies.net_kourier)" "${kourier_files[@]}"

#
# DOWNLOAD EVENTING
#
download_eventing eventing "$KNATIVE_EVENTING_VERSION" "${eventing_files[@]}"

