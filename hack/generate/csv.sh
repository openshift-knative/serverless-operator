#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

registry="quay.io/openshift-knative"
serving="${registry}/knative-serving"
serving_version="v$(metadata.get dependencies.serving)"
eventing="${registry}/knative-eventing"
eventing_version="v$(metadata.get dependencies.eventing)"
eventing_kafka="${registry}/knative-eventing-kafka"
eventing_kafka_version="v$(metadata.get dependencies.eventing_kafka)"
rbac_proxy="registry.ci.openshift.org/origin/4.7:kube-rbac-proxy"

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

image "queue-proxy"    "${serving}-queue:${serving_version}"
image "activator"      "${serving}-activator:${serving_version}"
image "autoscaler"     "${serving}-autoscaler:${serving_version}"
image "autoscaler-hpa" "${serving}-autoscaler-hpa:${serving_version}"
image "controller__controller"     "${serving}-controller"
image "webhook__webhook" "${serving}-webhook:${serving_version}"
image "domain-mapping" "${serving}-domain-mapping:${serving_version}"
image "domainmapping-webhook" "${serving}-domain-mapping-webhook:${serving_version}"
image "storage-version-migration-serving-serving-$(metadata.get dependencies.serving)__migrate" "${serving}-storage-version-migration:${serving_version}"

image "kourier-gateway" "quay.io/openshift-knative/proxyv2-ubi8:$(metadata.get dependencies.maistra)"
image "kourier-control" "${registry}/kourier:v$(metadata.get dependencies.kourier)"
image "net-kourier-controller__controller" "${registry}/kourier:v$(metadata.get dependencies.kourier)"

image "net-istio-controller__controller" "${registry}/net-istio-controller:v$(metadata.get dependencies.net_istio)"
image "net-istio-webhook__webhook" "${registry}/net-istio-webhook:v$(metadata.get dependencies.net_istio)"

image "eventing-controller__eventing-controller"    "${eventing}-controller:${eventing_version}"
image "sugar-controller__controller"                "${eventing}-sugar-controller:${eventing_version}"
image "eventing-webhook__eventing-webhook"          "${eventing}-webhook:${eventing_version}"
image "storage-version-migration-eventing-eventing-$(metadata.get dependencies.eventing)__migrate" "${eventing}-storage-version-migration:${eventing_version}"
image "mt-broker-controller__mt-broker-controller"  "${eventing}-mtchannel-broker:${eventing_version}"
image "mt-broker-filter__filter"                    "${eventing}-mtbroker-filter:${eventing_version}"
image "mt-broker-ingress__ingress"                  "${eventing}-mtbroker-ingress:${eventing_version}"
image "imc-controller__controller"                  "${eventing}-channel-controller:${eventing_version}"
image "imc-dispatcher__dispatcher"                  "${eventing}-channel-dispatcher:${eventing_version}"
image "pingsource-mt-adapter__dispatcher"           "${eventing}-mtping:${eventing_version}"

image "APISERVER_RA_IMAGE"   "${eventing}-apiserver-receive-adapter:${eventing_version}"
image "DISPATCHER_IMAGE"     "${eventing}-channel-dispatcher:${eventing_version}"

kafka_image "kafka-controller-manager__manager"    "${eventing_kafka}-source-controller:${eventing_kafka_version}"
kafka_image "KAFKA_RA_IMAGE"                       "${eventing_kafka}-receive-adapter:${eventing_kafka_version}"
kafka_image "kafka-ch-controller__controller"      "${eventing_kafka}-consolidated-controller:${eventing_kafka_version}"
kafka_image "DISPATCHER_IMAGE"                     "${eventing_kafka}-consolidated-dispatcher:${eventing_kafka_version}"
kafka_image "kafka-webhook__kafka-webhook"         "${eventing_kafka}-webhook:${eventing_kafka_version}"

image "KUBE_RBAC_PROXY"   "${rbac_proxy}"

declare -A yaml_keys
yaml_keys[spec.version]="$(metadata.get project.version)"
yaml_keys[metadata.name]="$(metadata.get project.name).v$(metadata.get project.version)"
yaml_keys['metadata.annotations[olm.skipRange]']="$(metadata.get olm.skipRange)"
yaml_keys[spec.minKubeVersion]="$(metadata.get requirements.kube.minVersion)"
yaml_keys[spec.replaces]="$(metadata.get project.name).v$(metadata.get olm.replaces)"

declare -A vars
vars[OCP_TARGET]="$(metadata.get 'requirements.ocpVersion.min')"

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
  path: spec.install.spec.deployments(name==knative-operator).spec.template.spec.containers(name==knative-operator).env[+]
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
add_downstream_operator_deployment_env "$target" "KNATIVE_EVENTING_KAFKA_VERSION" "$(metadata.get dependencies.eventing_kafka)"

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
