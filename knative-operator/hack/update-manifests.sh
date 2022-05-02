#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

kafka_controller_files=(eventing-kafka-controller.yaml eventing-kafka-post-install.yaml)
kafka_broker_files=(eventing-kafka-broker.yaml)
kafka_channel_files=(eventing-kafka-channel.yaml)
kafka_source_files=(eventing-kafka-source.yaml)
kafka_sink_files=(eventing-kafka-sink.yaml)
component_dir="$root/knative-operator/deploy/resources/knativekafka"

function download_kafka {
  subdir=$1
  shift

  files=("$@")

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

rm -rf "${component_dir}/controller"
rm -rf "${component_dir}/broker"
rm -rf "${component_dir}/channel"
rm -rf "${component_dir}/source"
rm -rf "${component_dir}/sink"

download_kafka controller "${kafka_controller_files[@]}"
download_kafka broker "${kafka_broker_files[@]}"
download_kafka channel "${kafka_channel_files[@]}"
download_kafka sink "${kafka_sink_files[@]}"
download_kafka source "${kafka_source_files[@]}"

# __Note__
# artifacts are downloaded from midstream openshift/release/artifacts directory.
# Before adding patches to this file consider sending a patch to midstream and then by just running
# `make generated-files` the patch will appear in the final bundled artifacts.
