#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

export CURRENT_VERSION_IMAGES=${CURRENT_VERSION_IMAGES:-"nightly"}

registry_host='registry.ci.openshift.org'
registry="${registry_host}/openshift"
client_version="$(metadata.get dependencies.cli)"
kn_event="${registry_host}/knative/release-${client_version%.*}:client-plugin-event"
rbac_proxy="registry.ci.openshift.org/origin/4.7:kube-rbac-proxy"

function default_knative_serving_images() {
  local serving
  serving="${registry}/knative-serving"
  local tag
  tag=$(metadata.get dependencies.serving)
  export KNATIVE_SERVING_QUEUE=${KNATIVE_SERVING_QUEUE:-"${serving}-queue:${tag}"}
  export KNATIVE_SERVING_ACTIVATOR=${KNATIVE_SERVING_ACTIVATOR:-"${serving}-activator:${tag}"}
  export KNATIVE_SERVING_AUTOSCALER=${KNATIVE_SERVING_AUTOSCALER:-"${serving}-autoscaler:${tag}"}
  export KNATIVE_SERVING_AUTOSCALER_HPA=${KNATIVE_SERVING_AUTOSCALER_HPA:-"${serving}-autoscaler-hpa:${tag}"}
  export KNATIVE_SERVING_CONTROLLER=${KNATIVE_SERVING_CONTROLLER:-"${serving}-controller:${tag}"}
  export KNATIVE_SERVING_WEBHOOK=${KNATIVE_SERVING_WEBHOOK:-"${serving}-webhook:${tag}"}
  export KNATIVE_SERVING_DOMAIN_MAPPING=${KNATIVE_SERVING_DOMAIN_MAPPING:-"${serving}-domain-mapping:${tag}"}
  export KNATIVE_SERVING_DOMAIN_MAPPING_WEBHOOK=${KNATIVE_SERVING_DOMAIN_MAPPING_WEBHOOK:-"${serving}-domain-mapping-webhook:${tag}"}
  export KNATIVE_SERVING_STORAGE_VERSION_MIGRATION=${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION:-"${serving}-storage-version-migration:${tag}"}
}

function default_knative_eventing_images() {
  local eventing
  eventing="${registry}/knative-eventing"
  local tag
  tag=$(metadata.get dependencies.eventing)
  export KNATIVE_EVENTING_CONTROLLER=${KNATIVE_EVENTING_CONTROLLER:-"${eventing}-controller:${tag}"}
  export KNATIVE_EVENTING_WEBHOOK=${KNATIVE_EVENTING_WEBHOOK:-"${eventing}-webhook:${tag}"}
  export KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION=${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION:-"${eventing}-storage-version-migration:${tag}"}
  export KNATIVE_EVENTING_MTBROKER_INGRESS=${KNATIVE_EVENTING_MTBROKER_INGRESS:-"${eventing}-mtbroker-ingress:${tag}"}
  export KNATIVE_EVENTING_MTBROKER_FILTER=${KNATIVE_EVENTING_MTBROKER_FILTER:-"${eventing}-mtbroker-filter:${tag}"}
  export KNATIVE_EVENTING_MTCHANNEL_BROKER=${KNATIVE_EVENTING_MTCHANNEL_BROKER:-"${eventing}-mtchannel-broker:${tag}"}
  export KNATIVE_EVENTING_MTPING=${KNATIVE_EVENTING_MTPING:-"${eventing}-mtping:${tag}"}
  export KNATIVE_EVENTING_CHANNEL_CONTROLLER=${KNATIVE_EVENTING_CHANNEL_CONTROLLER:-"${eventing}-channel-controller:${tag}"}
  export KNATIVE_EVENTING_CHANNEL_DISPATCHER=${KNATIVE_EVENTING_CHANNEL_DISPATCHER:-"${eventing}-channel-dispatcher:${tag}"}
  export KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER=${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER:-"${eventing}-apiserver-receive-adapter:${tag}"}
}

function default_knative_eventing_istio_images() {
  local eventing_istio
  eventing_istio="${registry}/knative-eventing-istio"
  local tag
  tag=$(metadata.get dependencies.eventing_istio)
  export KNATIVE_EVENTING_ISTIO_CONTROLLER=${KNATIVE_EVENTING_ISTIO_CONTROLLER:-"${eventing_istio}-controller:${tag}"}
}

function default_knative_eventing_kafka_broker_images() {
  local eventing_kafka_broker
  local tag
  tag=$(metadata.get dependencies.eventing_kafka_broker)
  eventing_kafka_broker="${registry}/knative-eventing-kafka-broker"
  export KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER=${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER:-"${eventing_kafka_broker}-dispatcher:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER=${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER:-"${eventing_kafka_broker}-receiver:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER=${KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER:-"${eventing_kafka_broker}-kafka-controller:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA=${KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA:-"${eventing_kafka_broker}-webhook-kafka":${tag}}
  export KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL=${KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL:-"${eventing_kafka_broker}-post-install:${tag}"}
}

function default_knative_ingress_images() {
  local knative_kourier knative_istio
  knative_kourier="$(metadata.get dependencies.kourier)"
  export KNATIVE_KOURIER_CONTROL=${KNATIVE_KOURIER_CONTROL:-"${registry}/net-kourier-kourier:${knative_kourier}"}
  export KNATIVE_KOURIER_GATEWAY=${KNATIVE_KOURIER_GATEWAY:-"quay.io/maistra-dev/proxyv2-ubi8:$(metadata.get dependencies.maistra)"}

  knative_istio="$(metadata.get dependencies.net_istio)"
  export KNATIVE_ISTIO_CONTROLLER="quay.io/rlehmann/main.go:latest"
  export KNATIVE_ISTIO_WEBHOOK=${KNATIVE_ISTIO_WEBHOOK:-"${registry}/net-istio-webhook:${knative_istio}"}
}

default_knative_eventing_images
default_knative_eventing_istio_images
default_knative_eventing_kafka_broker_images
default_knative_serving_images
default_knative_ingress_images

declare -a images
declare -A images_addresses

declare -a kafka_images
declare -A kafka_images_addresses

function image {
  local name address
  name="${1:?Pass a image name as arg[1]}"
  address="${2:?Pass a image address as arg[2]}"
  images+=("${name}")
  images_addresses["${name}"]="${address}"
}

function kafka_image {
  local name address
  name="${1:?Pass a image name as arg[1]}"
  address="${2:?Pass a image address as arg[2]}"
  kafka_images+=("${name}")
  kafka_images_addresses["${name}"]="${address}"
}

serving_version=$(metadata.get dependencies.serving)
serving_version=${serving_version/knative-v/}

image "queue-proxy"    "${KNATIVE_SERVING_QUEUE}"
image "activator"      "${KNATIVE_SERVING_ACTIVATOR}"
image "autoscaler"     "${KNATIVE_SERVING_AUTOSCALER}"
image "autoscaler-hpa" "${KNATIVE_SERVING_AUTOSCALER_HPA}"
image "controller__controller"     "${KNATIVE_SERVING_CONTROLLER}"
image "webhook__webhook" "${KNATIVE_SERVING_WEBHOOK}"
image "domain-mapping" "${KNATIVE_SERVING_DOMAIN_MAPPING}"
image "domainmapping-webhook" "${KNATIVE_SERVING_DOMAIN_MAPPING_WEBHOOK}"
image "storage-version-migration-serving-serving-${serving_version}__migrate" "${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION}"

image "kourier-gateway" "${KNATIVE_KOURIER_GATEWAY}"
image "net-kourier-controller__controller" "${KNATIVE_KOURIER_CONTROL}"

image "net-istio-controller__controller" "${KNATIVE_ISTIO_CONTROLLER}"
image "net-istio-webhook__webhook" "${KNATIVE_ISTIO_WEBHOOK}"

eventing_version=$(metadata.get dependencies.eventing)
eventing_version=${eventing_version/knative-v/}

image "eventing-controller__eventing-controller"                                 "${KNATIVE_EVENTING_CONTROLLER}"
image "eventing-istio-controller__eventing-istio-controller"                     "${KNATIVE_EVENTING_ISTIO_CONTROLLER}"
image "eventing-webhook__eventing-webhook"                                       "${KNATIVE_EVENTING_WEBHOOK}"
image "storage-version-migration-eventing-eventing-${eventing_version}__migrate" "${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION}"
image "mt-broker-controller__mt-broker-controller"                               "${KNATIVE_EVENTING_MTCHANNEL_BROKER}"
image "mt-broker-filter__filter"                                                 "${KNATIVE_EVENTING_MTBROKER_FILTER}"
image "mt-broker-ingress__ingress"                                               "${KNATIVE_EVENTING_MTBROKER_INGRESS}"
image "imc-controller__controller"                                               "${KNATIVE_EVENTING_CHANNEL_CONTROLLER}"
image "imc-dispatcher__dispatcher"                                               "${KNATIVE_EVENTING_CHANNEL_DISPATCHER}"
image "pingsource-mt-adapter__dispatcher"                                        "${KNATIVE_EVENTING_MTPING}"
image "APISERVER_RA_IMAGE"                                                       "${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER}"
image "DISPATCHER_IMAGE"                                                         "${KNATIVE_EVENTING_CHANNEL_DISPATCHER}"

kafka_image "kafka-broker-receiver__kafka-broker-receiver"       "${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER}"
kafka_image "kafka-broker-dispatcher__kafka-broker-dispatcher"   "${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER}"
kafka_image "kafka-channel-receiver__kafka-channel-receiver"     "${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER}"
kafka_image "kafka-channel-dispatcher__kafka-channel-dispatcher" "${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER}"
kafka_image "kafka-controller__controller"                       "${KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER}"
kafka_image "kafka-sink-receiver__kafka-sink-receiver"           "${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER}"
kafka_image "kafka-source-dispatcher__kafka-source-dispatcher"   "${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER}"
kafka_image "kafka-webhook-eventing__kafka-webhook-eventing"     "${KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA}"
kafka_image "kafka-controller-post-install__post-install"        "${KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL}"
kafka_image "knative-kafka-storage-version-migrator__migrate"    "${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION}" # Use eventing core image

image 'KUBE_RBAC_PROXY'          "${rbac_proxy}"
image 'KN_PLUGIN_EVENT_SENDER'   "${kn_event}-sender"
image 'KN_CLIENT'              "${registry}/knative-v$(metadata.get dependencies.cli):knative-client"

image 'KN_PLUGIN_FUNC_UTIL'           "$(metadata.get dependencies.func.util)"
image 'KN_PLUGIN_FUNC_TEKTON_S2I'     "$(metadata.get dependencies.func.tekton_s2i)"
image 'KN_PLUGIN_FUNC_TEKTON_BUILDAH' "$(metadata.get dependencies.func.tekton_buildah)"
image 'KN_PLUGIN_FUNC_NODEJS_16'      "$(metadata.get dependencies.func.nodejs_16)"
image 'KN_PLUGIN_FUNC_OPENJDK_17'     "$(metadata.get dependencies.func.openjdk_17)"

declare -A yaml_keys
yaml_keys[spec.version]="$(metadata.get project.version)"
yaml_keys[metadata.name]="$(metadata.get project.name).v$(metadata.get project.version)"
yaml_keys['metadata.annotations[olm.skipRange]']="$(metadata.get olm.skipRange)"
yaml_keys[spec.minKubeVersion]="$(metadata.get requirements.kube.minVersion)"
yaml_keys[spec.replaces]="$(metadata.get project.name).v$(metadata.get olm.replaces)"

declare -A vars
vars[OCP_TARGET]="$(metadata.get 'requirements.ocpVersion.max')"

function add_related_image {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.relatedImages[+]
  value:
    name: "${2}"
    image: "${3}"
EOF
}

function add_downstream_operator_deployment_env {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers(name==knative-openshift).env[+]
  value:
    name: "${2}"
    value: "${3}"
EOF
}

# since we also parse the environment variables in the upstream (actually midstream) operator,
# we don't add scope prefixes to image overrides here. We don't have a clash anyway without any scope prefixes!
# there was a naming clash between eventing and kafka, but we won't provide the Kafka overrides to the
# midstream operator.
function add_upstream_operator_deployment_env {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-operator-webhook).spec.template.spec.containers(name==knative-operator).env[+]
  value:
    name: "${2}"
    value: "${3}"
EOF
}

# Start fresh
cp "$template" "$target"

for name in "${images[@]}"; do
  echo "Image: ${name} -> ${images_addresses[$name]}"
  add_related_image "$target" "IMAGE_${name}" "${images_addresses[$name]}"
  add_downstream_operator_deployment_env "$target" "IMAGE_${name}" "${images_addresses[$name]}"
  add_upstream_operator_deployment_env "$target" "IMAGE_${name}" "${images_addresses[$name]}"
done

# don't add Kafka image overrides to upstream operator
for name in "${kafka_images[@]}"; do
  echo "kafka Image: ${name} -> ${kafka_images_addresses[$name]}"
  add_related_image "$target" "KAFKA_IMAGE_${name}" "${kafka_images_addresses[$name]}"
  add_downstream_operator_deployment_env "$target" "KAFKA_IMAGE_${name}" "${kafka_images_addresses[$name]}"
done

# Add Knative Kafka version to the downstream operator
add_downstream_operator_deployment_env "$target" "CURRENT_VERSION" "$(metadata.get project.version)"
ekb_version=$(metadata.get dependencies.eventing_kafka_broker)
add_downstream_operator_deployment_env "$target" "KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${ekb_version/knative-v/}" # Remove `knative-v` prefix if exists

# Add Serverless version to be used for naming storage jobs for Serving, Eventing
add_upstream_operator_deployment_env "$target" "CURRENT_VERSION" "$(metadata.get project.version)"

# Override the image for the CLI artifact deployment
yq write --inplace "$target" "spec.install.spec.deployments(name==knative-openshift).spec.template.spec.initContainers(name==cli-artifacts).image" "${registry}/knative-v$(metadata.get dependencies.cli):kn-cli-artifacts"

for name in "${!yaml_keys[@]}"; do
  echo "Value: ${name} -> ${yaml_keys[$name]}"
  yq write --inplace "$target" "$name" "${yaml_keys[$name]}"
done

for name in "${!vars[@]}"; do
  echo "Value: ${name} -> ${vars[$name]}"
  sed --in-place "s/__${name}__/${vars[${name}]}/" "$target"
done

echo "CURRENT_VERSION_IMAGES ${CURRENT_VERSION_IMAGES}"

# Replace operator images reference based on CURRENT_VERSION_IMAGES env variable
temp_csv="${target}.tmp"
# Variable is expected to not be expended, so disable shellcheck check.
# shellcheck disable=SC2016
envsubst '$CURRENT_VERSION_IMAGES' <"${target}" >"${temp_csv}"
mv "${temp_csv}" "${target}"
