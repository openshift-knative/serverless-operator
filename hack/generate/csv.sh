#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.bash"
# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"
# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/images.bash"

# TODO migrate CLI images to Konflux
client_version="$(metadata.get dependencies.cli)"
kn_event="${ci_registry_host}/knative/release-${client_version#knative-v}:client-plugin-event"

rbac_proxy=$(metadata.get 'dependencies.kube_rbac_proxy')

default_serverless_operator_images
default_knative_ingress_images

if [[ ${USE_RELEASE_NEXT:-} == "true" ]]; then
  knative_eventing_images_release
  knative_eventing_istio_images_release
  knative_eventing_kafka_broker_images_release
  knative_backstage_plugins_images_release
  knative_serving_images_release
  knative_kn_plugin_func_images_release
else
  default_knative_eventing_images
  default_knative_eventing_istio_images
  default_knative_eventing_kafka_broker_images
  default_knative_backstage_plugins_images
  default_knative_serving_images
  default_knative_kn_plugin_func_images
fi

declare -a operator_images
declare -A operator_images_addresses

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

function operator_image {
  local name address
  name="${1:?Pass a image name as arg[1]}"
  address="${2:?Pass a image address as arg[2]}"
  operator_images+=("${name}")
  operator_images_addresses["${name}"]="${address}"
}

operator_image "knative-operator" "${SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR}"
operator_image "knative-openshift" "${SERVERLESS_KNATIVE_OPERATOR}"
operator_image "knative-openshift-ingress" "${SERVERLESS_INGRESS}"

serving_version=$(metadata.get dependencies.serving)
serving_version=${serving_version/knative-v/}

image "queue-proxy"    "${KNATIVE_SERVING_QUEUE}"
image "activator"      "${KNATIVE_SERVING_ACTIVATOR}"
image "autoscaler"     "${KNATIVE_SERVING_AUTOSCALER}"
image "autoscaler-hpa" "${KNATIVE_SERVING_AUTOSCALER_HPA}"
image "controller__controller"     "${KNATIVE_SERVING_CONTROLLER}"
image "webhook__webhook" "${KNATIVE_SERVING_WEBHOOK}"
image "storage-version-migration-serving-__migrate" "${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION}"

image "kourier-gateway" "${KNATIVE_KOURIER_GATEWAY}"
image "net-kourier-controller__controller" "${KNATIVE_KOURIER_CONTROL}"

image "net-istio-controller__controller" "${KNATIVE_ISTIO_CONTROLLER}"
image "net-istio-webhook__webhook" "${KNATIVE_ISTIO_WEBHOOK}"

eventing_version=$(metadata.get dependencies.eventing)
eventing_version=${eventing_version/knative-v/}

image "eventing-controller__eventing-controller"                                 "${KNATIVE_EVENTING_CONTROLLER}"
image "eventing-istio-controller__eventing-istio-controller"                     "${KNATIVE_EVENTING_ISTIO_CONTROLLER}"
image "eventing-webhook__eventing-webhook"                                       "${KNATIVE_EVENTING_WEBHOOK}"
image "storage-version-migration-eventing-__migrate"                             "${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION}"
image "mt-broker-controller__mt-broker-controller"                               "${KNATIVE_EVENTING_MTCHANNEL_BROKER}"
image "mt-broker-filter__filter"                                                 "${KNATIVE_EVENTING_FILTER}"
image "mt-broker-ingress__ingress"                                               "${KNATIVE_EVENTING_INGRESS}"
image "imc-controller__controller"                                               "${KNATIVE_EVENTING_CHANNEL_CONTROLLER}"
image "imc-dispatcher__dispatcher"                                               "${KNATIVE_EVENTING_CHANNEL_DISPATCHER}"
image "pingsource-mt-adapter__dispatcher"                                        "${KNATIVE_EVENTING_MTPING}"
image "APISERVER_RA_IMAGE"                                                       "${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER}"
image "DISPATCHER_IMAGE"                                                         "${KNATIVE_EVENTING_CHANNEL_DISPATCHER}"
if [ "${KNATIVE_EVENTING_JOBSINK}" != "" ]; then
  image "job-sink__job-sink"                                                       "${KNATIVE_EVENTING_JOBSINK}"
fi

image "eventmesh-backend__controller" "${KNATIVE_BACKSTAGE_PLUGINS_EVENTMESH}"

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
image 'KN_CLIENT'                "${ci_registry}/$(metadata.get dependencies.cli):knative-client-kn"

image "KN_PLUGIN_FUNC_UTIL"               "${KNATIVE_KN_PLUGIN_FUNC_FUNC_UTIL}"
image "KN_PLUGIN_FUNC_TEKTON_S2I"         "${KNATIVE_KN_PLUGIN_FUNC_TEKTON_S2I}"
image "KN_PLUGIN_FUNC_TEKTON_BUILDAH"     "${KNATIVE_KN_PLUGIN_FUNC_TEKTON_BUILDAH}"
image "KN_PLUGIN_FUNC_NODEJS_20_MINIMAL"  "${KNATIVE_KN_PLUGIN_FUNC_NODEJS_20_MINIMAL}"
image "KN_PLUGIN_FUNC_OPENJDK_21"         "${KNATIVE_KN_PLUGIN_FUNC_OPENJDK_21}"
image "KN_PLUGIN_FUNC_PYTHON_39"          "${KNATIVE_KN_PLUGIN_FUNC_PYTHON_39}"

declare -A yaml_keys
yaml_keys[spec.version]="$(metadata.get project.version)"
yaml_keys[metadata.name]="$(metadata.get project.name).v$(metadata.get project.version)"
yaml_keys['metadata.annotations[olm.skipRange]']="$(metadata.get olm.skipRange)"
yaml_keys['metadata.annotations[operators.openshift.io/must-gather-image]']="$(metadata.get dependencies.mustgather.image)"
yaml_keys[spec.minKubeVersion]="$(metadata.get requirements.kube.minVersion)"
yaml_keys[spec.replaces]="$(metadata.get project.name).v$(metadata.get olm.replaces)"

declare -A vars
vars[OCP_TARGET]="$(metadata.get 'requirements.ocpVersion.list[-1]')"
vars[VERSION_MAJOR_MINOR]="$(versions.major_minor $(metadata.get 'project.version'))"

function add_related_image {
  echo "Add related image to '${1}' - $2 = $3"
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.relatedImages[+]
  value:
    name: "${2}"
    image: "${3}"
EOF
}

function add_downstream_operator_deployment_env {
  echo "Add downstream operator deployment env '${1}' - $2 = $3"
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers(name==knative-openshift).env[+]
  value:
    name: "${2}"
    value: "${3}"
EOF
}

function set_operator_downstream_image {
  yq write --inplace "$1" "spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers(name==knative-openshift).image" "${SERVERLESS_KNATIVE_OPERATOR}"
}

function set_operator_upstream_image {
  yq write --inplace "$1" "spec.install.spec.deployments(name==knative-operator-webhook).spec.template.spec.containers(name==knative-operator).image" "${SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR}"
}

function set_operator_ingress_image {
  yq write --inplace "$1" "spec.install.spec.deployments(name==knative-openshift-ingress).spec.template.spec.containers(name==knative-openshift-ingress).image" "${SERVERLESS_INGRESS}"
}

# since we also parse the environment variables in the upstream (actually midstream) operator,
# we don't add scope prefixes to image overrides here. We don't have a clash anyway without any scope prefixes!
# there was a naming clash between eventing and kafka, but we won't provide the Kafka overrides to the
# midstream operator.
function add_upstream_operator_deployment_env {
  echo "Add upstream operator deployment env '${1}' - $2 = $3"
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

set_operator_upstream_image "$target"
set_operator_downstream_image "$target"
set_operator_ingress_image "$target"

for name in "${operator_images[@]}"; do
  echo "Image: ${name} -> ${operator_images_addresses[$name]}"
  add_related_image "$target" "${name}" "${operator_images_addresses[$name]}"
done

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

add_related_image "$target" "IMAGE_MUST_GATHER" "$(metadata.get dependencies.mustgather.image)"

# Add Knative Kafka version to the downstream operator
add_downstream_operator_deployment_env "$target" "CURRENT_VERSION" "$(metadata.get project.version)"
serving_version=$(metadata.get dependencies.serving)
add_downstream_operator_deployment_env "$target" "KNATIVE_SERVING_VERSION" "${serving_version/knative-v/}" # Remove `knative-v` prefix if exists
eventing_version=$(metadata.get dependencies.eventing)
add_downstream_operator_deployment_env "$target" "KNATIVE_EVENTING_VERSION" "${eventing_version/knative-v/}" # Remove `knative-v` prefix if exists
ekb_version=$(metadata.get dependencies.eventing_kafka_broker)
add_downstream_operator_deployment_env "$target" "KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${ekb_version/knative-v/}" # Remove `knative-v` prefix if exists

# Add Serverless version to be used for naming storage jobs for Serving, Eventing
add_upstream_operator_deployment_env "$target" "CURRENT_VERSION" "$(metadata.get project.version)"
add_upstream_operator_deployment_env "$target" "KNATIVE_SERVING_VERSION" "${serving_version/knative-v/}" # Remove `knative-v` prefix if exists
add_upstream_operator_deployment_env "$target" "KNATIVE_EVENTING_VERSION" "${eventing_version/knative-v/}" # Remove `knative-v` prefix if exists
add_upstream_operator_deployment_env "$target" "KNATIVE_EVENTING_KAFKA_BROKER_VERSION" "${ekb_version/knative-v/}" # Remove `knative-v` prefix if exists

# Override the image for the CLI artifact deployment
yq write --inplace "$target" "spec.install.spec.deployments(name==knative-openshift).spec.template.spec.initContainers(name==cli-artifacts).image" "${ci_registry}/$(metadata.get dependencies.cli):knative-client-cli-artifacts"

for name in "${!yaml_keys[@]}"; do
  echo "Value: ${name} -> ${yaml_keys[$name]}"
  yq write --inplace "$target" "$name" "${yaml_keys[$name]}"
done

for name in "${!vars[@]}"; do
  echo "Value: ${name} -> ${vars[$name]}"
  sed --in-place "s/__${name}__/${vars[${name}]}/" "$target"
done
