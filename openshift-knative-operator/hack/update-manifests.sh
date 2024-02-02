#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# These files could in theory change from release to release, though their names should
# be fairly stable.
serving_files=(serving-crds serving-core serving-hpa serving-post-install-jobs)
eventing_files=(eventing-crds.yaml eventing-core.yaml in-memory-channel.yaml mt-channel-broker.yaml eventing-post-install.yaml eventing-tls-networking.yaml)
eventing_istio_files=(eventing-istio-controller.yaml)

# This excludes the gateways and peerauthentication settings as we want customers to do
# manipulate those.
istio_files=(net-istio-core)

kourier_files=(net-kourier)

export KNATIVE_EVENTING_MANIFESTS_DIR=${KNATIVE_EVENTING_MANIFESTS_DIR:-""}
export KNATIVE_EVENTING_ISTIO_MANIFESTS_DIR=${KNATIVE_EVENTING_ISTIO_MANIFESTS_DIR:-""}
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
  target_dir="${component_dir}/${version/knative-v/}" # remove `knative-v` prefix
  rm -r "$component_dir"
  mkdir -p "$target_dir"

  branch=$(metadata.get dependencies.serving_artifacts_branch)
  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"

    if [[ ${KNATIVE_SERVING_MANIFESTS_DIR} = "" ]]; then
      url="https://raw.githubusercontent.com/openshift-knative/serving/${branch}/openshift/release/artifacts/$file"
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

      url="https://raw.githubusercontent.com/openshift-knative/eventing/${branch}/openshift/release/artifacts/$file"
      echo "Downloading file from ${url}"
      wget --no-check-certificate "$url" -O "$target_file"
    else
      cp "${KNATIVE_EVENTING_MANIFESTS_DIR}/${file}" "$target_file"
    fi

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

function download_eventing_istio {
  component=$1
  version=$2
  shift
  shift

  files=("$@")
  echo "Files: ${files[*]}"

  component_dir="$root/openshift-knative-operator/cmd/operator/kodata/knative-${component}"
  target_dir="${component_dir}/${version/knative-v/}" # remove `knative-v` prefix

  branch=$(metadata.get dependencies.eventing_istio_artifacts_branch)
  for ((i = 0; i < ${#files[@]}; i++)); do
    index=$(( i+1 ))
    file="${files[$i]}"
    target_file="$target_dir/$index-$file"
    if [[ ${KNATIVE_EVENTING_ISTIO_MANIFESTS_DIR} = "" ]]; then

      url="https://raw.githubusercontent.com/openshift-knative/eventing-istio/${branch}/openshift/release/artifacts/$file"
      echo "Downloading file from ${url}"
      wget --no-check-certificate "$url" -O "$target_file"
    else
      cp "${KNATIVE_EVENTING_ISTIO_MANIFESTS_DIR}/${file}" "$target_file"
    fi

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

function download_ingress {
  component=$1
  ingress_dir=$2
  subdir=$3
  shift
  shift
  shift

  mkdir -p "$ingress_dir/$subdir"

  files=("$@")
  echo "Files: ${files[*]}"

  branch=$(metadata.get "dependencies.${component/-/_}_artifacts_branch")
  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    ingress_target_file="$ingress_dir/$subdir/$index-$file"

    url="https://raw.githubusercontent.com/openshift-knative/${component}/${branch}/openshift/release/artifacts/$file"

    wget --no-check-certificate "$url" -O "$ingress_target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$ingress_target_file"
  done
}

function download_eventing_tls_testing_resources {

  files=("$@")
  echo "Files: ${files[*]}"

  component_dir="$root/hack/lib/certmanager_resources"
  target_dir="${component_dir}"

  branch=$(metadata.get dependencies.eventing_artifacts_branch)
  for ((i = 0; i < ${#files[@]}; i++)); do
    index=$(( i+1 ))
    file="${files[$i]}"
    target_file="$target_dir/$file"

    url="https://raw.githubusercontent.com/openshift-knative/eventing/${branch}/openshift/tls/issuers/$file"
    echo "Downloading file from ${url}"
    wget --no-check-certificate "$url" -O "$target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

#
# DOWNLOAD SERVING
#

# When openshift-knative/serving uses this repo to run a job (eg. PR against openshift-knative/serving) it will use a minimum
# setup with net-kourier. Thus it will not use the release artifacts generated under openshift-knative-operator/cmd/kodata/knative-serving.
# Instead openshift-knative/serving uses its own generated ci manifests and sets KNATIVE_SERVING_TEST_MANIFESTS_DIR.
# Extensive Serving testing is done at this repo only. For the latter we do use manifests under openshift-knative-operator/cmd/kodata/knative-serving which are fetched from the midstream
# repo.
if [[ ${KNATIVE_SERVING_TEST_MANIFESTS_DIR} = "" ]]; then
  download_serving serving "${KNATIVE_SERVING_VERSION}" "${serving_files[@]}"
fi

#
# DOWNLOAD INGRESS
#

# clean up ingreess_dir
ingress_root_dir="$root/openshift-knative-operator/cmd/operator/kodata/ingress/"
rm -rf "${ingress_root_dir}"

serving_version=$(versions.major_minor "${KNATIVE_SERVING_VERSION}")
ingress_dir="${ingress_root_dir}/${serving_version/knative-v/}" # remove `knative-v` prefix
mkdir -p "${ingress_dir}"

# ingress_dir has to contain a sub folder for each ingress
# that corresponds to the string in https://github.com/knative/operator/blob/main/pkg/reconciler/knativeserving/ingress/ingress.go#L76
download_ingress net-istio "${ingress_dir}" "istio" "${istio_files[@]}"
download_ingress net-kourier "${ingress_dir}" "kourier" "${kourier_files[@]}"

#
# DOWNLOAD EVENTING
#
download_eventing eventing "$KNATIVE_EVENTING_VERSION" "${eventing_files[@]}"
download_eventing_istio eventing "$KNATIVE_EVENTING_VERSION" "${eventing_istio_files[@]}"

download_eventing_tls_testing_resources "ca-certificate.yaml" "eventing-ca-issuer.yaml" "selfsigned-issuer.yaml"

