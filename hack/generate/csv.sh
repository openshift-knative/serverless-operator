#!/bin/bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

registry="registry.svc.ci.openshift.org/openshift"
serving="${registry}/knative-v$(metadata.get dependencies.serving):knative-serving"
eventing="${registry}/knative-v$(metadata.get dependencies.eventing):knative-eventing"
eventing_contrib="${registry}/knative-v$(metadata.get dependencies.eventing_contrib):knative-eventing-sources"

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

image "queue-proxy"    "${serving}-queue"
image "activator"      "${serving}-activator"
image "autoscaler"     "${serving}-autoscaler"
image "autoscaler-hpa" "${serving}-autoscaler-hpa"
image "controller"     "${serving}-controller"
image "webhook"        "${serving}-webhook"
image "storage-version-migration-serving-$(metadata.get dependencies.serving)__migrate" "${serving}-storage-version-migration"

image "3scale-kourier-gateway" "docker.io/maistra/proxyv2-ubi8:$(metadata.get dependencies.maistra)"
image "3scale-kourier-control" "${registry}/knative-v$(metadata.get dependencies.kourier):kourier"

image "eventing-controller__eventing-controller"    "${eventing}-controller"
image "sugar-controller__controller"                "${eventing}-sugar-controller"
image "eventing-webhook__eventing-webhook"          "${eventing}-webhook"
image "storage-version-migration-eventing__migrate" "${eventing}-storage-version-migration"
image "mt-broker-controller__mt-broker-controller"  "${eventing}-mtchannel-broker"
image "mt-broker-filter__filter"                    "${eventing}-mtbroker-filter"
image "mt-broker-ingress__ingress"                  "${eventing}-mtbroker-ingress"
image "imc-controller__controller"                  "${eventing}-channel-controller"
image "imc-dispatcher__dispatcher"                  "${eventing}-channel-dispatcher"

image "v0.17.0-pingsource-cleanup-eventing-$(metadata.get dependencies.eventing)__pingsource" "${eventing}-pingsource-cleanup"
image "PING_IMAGE"           "${eventing}-ping"
image "MT_PING_IMAGE"        "${eventing}-mtping"
image "APISERVER_RA_IMAGE"   "${eventing}-apiserver-receive-adapter"
image "DISPATCHER_IMAGE"     "${eventing}-channel-dispatcher"
image "KN_CLI_ARTIFACTS"     "${registry}/knative-v$(metadata.get dependencies.cli):kn-cli-artifacts"

kafka_image "kafka-controller-manager__manager"    "${eventing_contrib}-kafka-source-controller"
kafka_image "KAFKA_RA_IMAGE"                       "${eventing_contrib}-kafka-source-adapter"
kafka_image "kafka-ch-controller__controller"      "${eventing_contrib}-kafka-channel-controller"
kafka_image "DISPATCHER_IMAGE"                     "${eventing_contrib}-kafka-channel-dispatcher"
kafka_image "kafka-ch-dispatcher__dispatcher"      "${eventing_contrib}-kafka-channel-dispatcher"
kafka_image "kafka-webhook__kafka-webhook"         "${eventing_contrib}-kafka-channel-webhook"

declare -A yaml_keys
yaml_keys[spec.version]="$(metadata.get project.version)"
yaml_keys[metadata.name]="$(metadata.get project.name).v$(metadata.get project.version)"
yaml_keys['metadata.annotations[olm.skipRange]']="$(metadata.get olm.skipRange)"
yaml_keys[spec.minKubeVersion]="$(metadata.get requirements.kube.minVersion)"
yaml_keys[spec.replaces]="$(metadata.get project.name).v$(metadata.get olm.replaces)"

declare -A vars
vars[OCP_TARGET]="$(metadata.get 'requirements.ocp.[0]')"

function add_related_image {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.relatedImages[+]
  value:
    name: "${2}"
    image: "${3}"
EOF
}

function add_downstream_operator_deployment_image {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers[0].env[+]
  value:
    name: "${2}"
    value: "${3}"
EOF
}

# since we also parse the environment variables in the upstream (actually midstream) operator,
# we don't add scope prefixes to image overrides here. We don't have a clash anyway without any scope prefixes!
# there was a naming clash between eventing and kafka, but we won't provide the Kafka overrides to the
# midstream operator.
function add_upstream_operator_deployment_image {
  cat << EOF | yq write --inplace --script - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-operator).spec.template.spec.containers[0].env[+]
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
  add_downstream_operator_deployment_image "$target" "IMAGE_${name}" "${images_addresses[$name]}"
  add_upstream_operator_deployment_image "$target" "IMAGE_${name}" "${images_addresses[$name]}"
done

# don't add Kafka image overrides to upstream operator
for name in "${kafka_images[@]}"; do
  echo "kafka Image: ${name} -> ${kafka_images_addresses[$name]}"
  add_related_image "$target" "KAFKA_IMAGE_${name}" "${kafka_images_addresses[$name]}"
  add_downstream_operator_deployment_image "$target" "KAFKA_IMAGE_${name}" "${kafka_images_addresses[$name]}"
done

for name in "${!yaml_keys[@]}"; do
  echo "Value: ${name} -> ${yaml_keys[$name]}"
  yq write --inplace "$target" "$name" "${yaml_keys[$name]}"
done

for name in "${!vars[@]}"; do
  echo "Value: ${name} -> ${vars[$name]}"
  sed --in-place "s/__${name}__/${vars[${name}]}/" "$target"
done
