#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/images.bash"

default_knative_eventing_images
default_knative_eventing_istio_images
default_knative_eventing_kafka_broker_images
default_knative_backstage_plugins_images
default_knative_serving_images
default_knative_ingress_images
default_knative_kn_plugin_func_images
default_knative_client_images

envsubst < "$template" > "$target"
