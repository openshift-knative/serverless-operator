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
  done
}


version=${KOURIER_VERSION:-v$(metadata.get dependencies.kourier)}

target_dir="$root/knative-operator/deploy/resources/kourier"
rm -rf "$target_dir"
mkdir -p "$target_dir"

target_file="$target_dir/kourier-latest.yaml"

url="https://github.com/knative-sandbox/net-kourier/releases/download/$version/kourier.yaml"
wget --no-check-certificate "$url" -O "$target_file"

# TODO: [SRVKS-610] These values should be replaced by operator instead of sed.
sed -i -e 's/kourier-control.knative-serving/kourier-control.knative-serving-ingress/g' "$target_file"

download_kafka knativekafka "$KNATIVE_EVENTING_KAFKA_VERSION" "${kafka_files[@]}"
# For Backport of v1alpha1 hacks, we change the storage versions:
git apply "$root/knative-operator/hack/006-kafkachannel-storage-beta1.patch"

# This is for  SRVKe-807.
git apply "$root/knative-operator/hack/007-eventing-kafka-pdb.patch"
