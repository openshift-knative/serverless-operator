#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds.yaml eventing-core.yaml in-memory-channel.yaml mt-channel-broker.yaml eventing-sugar-controller.yaml eventing-post-install.yaml)

# This excludes the gateways and peerauthentication settings as we want customers to do
# manipulate those.
istio_files=(200-clusterrole 400-config-istio 500-controller 500-webhook-deployment 500-webhook-secret 500-webhook-service 600-mutating-webhook 600-validating-webhook)

export KNATIVE_EVENTING_MANIFESTS_DIR=${KNATIVE_EVENTING_MANIFESTS_DIR:-""}
export KNATIVE_SERVING_MANIFESTS_DIR=${KNATIVE_SERVING_MANIFESTS_DIR:-""}
export KNATIVE_SERVING_TEST_MANIFESTS_DIR=${KNATIVE_SERVING_TEST_MANIFESTS_DIR:-""}

function download_serving {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

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
      url="https://raw.githubusercontent.com/skonto/serving/${branch}/openshift/release/artifacts/$index-$file"
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
  target_dir="${component_dir}/${version:1}"
  rm -r "$component_dir"
  mkdir -p "$target_dir"

  for ((i = 0; i < ${#files[@]}; i++)); do
    index=$(( i+1 ))
    file="${files[$i]}"
    target_file="$target_dir/$index-$file"
    if [[ ${KNATIVE_EVENTING_MANIFESTS_DIR} = "" ]]; then
      branch=$(metadata.get dependencies.eventing_artifacts_branch)
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

  ingress_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
  rm -r "$ingress_dir"
  mkdir -p "$ingress_dir"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    ingress_target_file="$ingress_dir/$index-$file"
    url="https://raw.githubusercontent.com/knative-sandbox/${component}/knative-${version}/config/${file}"

    wget --no-check-certificate "$url" -O "$ingress_target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$ingress_target_file"
  done
}

#
# DOWNLOAD SERVING
#

# When openshift-knative/serving uses this repo to run a job (eg. PR against openshift-knative/serving) it will use a minimum
# setup with net-kourier. Thus it will not use the release artifacts generated under openshift-knative-operator/cmd/kodata/knative-serving.
# Instead openshift-knative/serving uses its own generated ci manifests and sets KNATIVE_SERVING_TEST_MANIFESTS_DIR.
# Extensive Serving testing is done at this repo only. For the latter we do use manifests under openshift-knative-operator/cmd/kodata/knative-serving which are fetched from the midstream
# repo. TODO: unify the artifacts at the mid stream repo.
if [[ ${KNATIVE_SERVING_TEST_MANIFESTS_DIR} = "" ]]; then
  download_serving serving "${KNATIVE_SERVING_VERSION}" "${serving_files[@]}"
fi

#
# DOWNLOAD INGRESS
#

download_ingress net-istio "v$(metadata.get dependencies.net_istio)" "${istio_files[@]}"

url="https://github.com/knative-sandbox/net-kourier/releases/download/knative-v$(metadata.get dependencies.kourier)/kourier.yaml"
kourier_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/$(versions.major_minor "${KNATIVE_SERVING_VERSION}")"
kourier_file="$kourier_dir/0-kourier.yaml"
wget --no-check-certificate "$url" -O "$kourier_file"
# TODO: [SRVKS-610] These values should be replaced by operator instead of sed.
sed -i -e 's/net-kourier-controller.knative-serving/net-kourier-controller/g' "$kourier_file"
# Break all image references so we know our overrides work correctly.
yaml.break_image_references "$kourier_file"
# Download config-network.yaml for Kourier. This is necessary as kourier uses different namespace (knative-serving-ingress).
config_network_url="https://raw.githubusercontent.com/knative/networking/release-$(versions.major_minor "${KNATIVE_SERVING_VERSION}")/config/config-network.yaml"
config_network="$kourier_dir/1-config-network.yaml"
wget --no-check-certificate "$config_network_url" -O "$config_network"
sed -i -e '/labels:$/a \    app.kubernetes.io\/component: kourier' "$config_network"
sed -i -e '/labels:$/a \    networking.knative.dev\/ingress-provider: kourier' "$config_network"

# Add networkpolicy for webhook when net-istio is enabled.
git apply "$root/openshift-knative-operator/hack/007-networkpolicy-mesh.patch"

# Make Kourier rollout in a more defensive way so no requests get dropped.
# TODO: Can probably be removed in 1.21 and/or be sent upstream.
git apply "$root/openshift-knative-operator/hack/008-kourier-rollout.patch"

#
# DOWNLOAD EVENTING
#
download_eventing eventing "$KNATIVE_EVENTING_VERSION" "${eventing_files[@]}"

# __Note__
# artifacts are downloaded from midstream openshift/release/artifacts directory.
# Before adding patches to this file consider sending a patch to midstream and then by just running
# `make generated-files` the patch will appear in the final bundled artifacts.
