#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

# legacy
kafka_channel_files=(channel-consolidated channel-post-install)

kafka_controller_files=(eventing-kafka-controller.yaml eventing-kafka-post-install.yaml)
kafka_broker_files=(eventing-kafka-broker.yaml)
kafka_source_files=(eventing-kafka-source.yaml)
kafka_sink_files=(eventing-kafka-sink.yaml)

function download_legacy_kafka {
  component=$1
  subdir=$2
  version=$3
  shift
  shift
  shift

  files=("$@")

  component_dir="$root/knative-operator/deploy/resources/knativekafka"
  target_dir="${component_dir}"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$subdir/$index-$file"
    url="https://github.com/knative-sandbox/$component/releases/download/knative-$version/$file"

    wget --no-check-certificate "$url" -O "$target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

download_legacy_kafka eventing-kafka channel "$KNATIVE_EVENTING_KAFKA_VERSION" "${kafka_channel_files[@]}"

# For 1.17 we still skip HPA
git apply "$root/knative-operator/hack/001-eventing-kafka-remove_hpa.patch"

# SRVKE-919: Change the minavailable pdb for kafka-webhook to 0
git apply "$root/knative-operator/hack/007-eventing-kafka-patch-pdb.patch"

# Fix for SRVKE-1171
git apply "$root/knative-operator/hack/011-eventing-kafkachannel-dead-letter-sink-uri.patch"
git apply "$root/knative-operator/hack/012-eventing-kafkachannel-addressable-resolver-binding.patch"

function download_kafka {

  subdir=$1
  shift

  files=("$@")
  component_dir="$root/knative-operator/deploy/resources/knativekafka"
  rm -rf "${component_dir:?}/${subdir}"
  mkdir -p "${component_dir:?}/${subdir}"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    file="${files[$i]}"
    target_file="$component_dir/$subdir/$file"
    branch=$(metadata.get dependencies.eventing_kafka_broker_artifacts_branch)
    url="https://raw.githubusercontent.com/openshift-knative/eventing-kafka-broker/${branch}/openshift/release/artifacts/$file"

    echo "Downloading file from ${url}"

    wget --no-check-certificate "$url" -O "$target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}


download_kafka controller "${kafka_controller_files[@]}"
download_kafka broker "${kafka_broker_files[@]}"
download_kafka sink "${kafka_sink_files[@]}"
download_kafka source "${kafka_source_files[@]}"

# For now we remove the CRDs, since the "broker" does not yet do anything with them
git apply "$root/knative-operator/hack/003-broker-remove-duplicated-crds.patch"
