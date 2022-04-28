#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

kafka_channel_files=(channel-consolidated channel-post-install)
kafka_source_files=(eventing-kafka-source)
kafka_controller_files=(eventing-kafka-controller)
kafka_broker_files=(eventing-kafka-broker)
kafka_sink_files=(eventing-kafka-sink)

function download_kafka {
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

download_kafka eventing-kafka channel "$KNATIVE_EVENTING_KAFKA_VERSION" "${kafka_channel_files[@]}"

# For 1.17 we still skip HPA
git apply "$root/knative-operator/hack/001-eventing-kafka-remove_hpa.patch"

# SRVKE-919: Change the minavailable pdb for kafka-webhook to 0
git apply "$root/knative-operator/hack/007-eventing-kafka-patch-pdb.patch"

# Kafka Broker content:
# Control-Plane files:
download_kafka eventing-kafka-broker controller "$KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${kafka_controller_files[@]}"

#Data-Plane Files Broker:
download_kafka eventing-kafka-broker broker "$KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${kafka_broker_files[@]}"

#Data-Plane Files Sink:
download_kafka eventing-kafka-broker sink "$KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${kafka_sink_files[@]}"

#Data-Plane Files Source:
download_kafka eventing-kafka-broker source "$KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${kafka_source_files[@]}"

# That CM is already there, with Eventing
git apply "$root/knative-operator/hack/001-broker-config-tracing.patch"

# For now we remove the CRDs, since the "broker" does not yet do anything with them
git apply "$root/knative-operator/hack/003-broker-remove-duplicated-crds.patch"

# Remove the config for the new, unused channel
git apply "$root/knative-operator/hack/004-remove_new_channel_cfg.patch"

# Fix for SRVKE-1171
git apply "$root/knative-operator/hack/011-eventing-kafkachannel-dead-letter-sink-uri.patch"
git apply "$root/knative-operator/hack/012-eventing-kafkachannel-addressable-resolver-binding.patch"

# SRVKE-1184 Migration to Broker components
git apply "$root/knative-operator/hack/014-eventing-kafka-lease-name-remapping.patch"

# SRVKE-1202 temporary fix for a missing ConfigMap knative-eventing/config-kafka-source-defaults
git apply "$root/knative-operator/hack/017-eventing-kafka-configmap-knative-eventing-config-kafka-source-defaults.patch"

# SRVKE-1162: potential fix
git apply "$root/knative-operator/hack/016-eventing-kafka-dispatcher-probes.patch"
