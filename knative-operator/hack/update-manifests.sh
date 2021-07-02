#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

kafka_files=(channel-consolidated source)

function download_kafka {
  component=$1
  version=$2
  shift
  shift

  files=("$@")

  component_dir="$root/knative-operator/deploy/resources/${component}"
  target_dir="${component_dir}"

  for (( i=0; i<${#files[@]}; i++ ));
  do
    index=$(( i+1 ))
    file="${files[$i]}.yaml"
    target_file="$target_dir/$index-$file"
    url="https://github.com/knative-sandbox/eventing-kafka/releases/download/$version/$file"

    wget --no-check-certificate "$url" -O "$target_file"

    # Break all image references so we know our overrides work correctly.
    yaml.break_image_references "$target_file"
  done
}

download_kafka knativekafka "$KNATIVE_EVENTING_KAFKA_VERSION" "${kafka_files[@]}"

# Change the minavailable pdb for kafka-webhook to 1
git apply "$root/knative-operator/hack/007-eventing-kafka-pdb.patch"
