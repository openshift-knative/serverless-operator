#!/bin/bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

source "$(dirname "${BASH_SOURCE[0]}")/../../hack/lib/vars.bash"

registry="registry.svc.ci.openshift.org/openshift"
serving="${registry}/knative-$KNATIVE_SERVING_VERSION:knative-serving"
eventing="${registry}/knative-$KNATIVE_EVENTING_VERSION:knative-eventing"
eventing_contrib="${registry}/knative-$KNATIVE_EVENTING_CONTRIB_VERSION:knative-eventing-sources"

declare -A serving_images
serving_images=(
  ["queue-proxy"]="${serving}-queue"
  ["activator"]="${serving}-activator"
  ["autoscaler"]="${serving}-autoscaler"
  ["autoscaler-hpa"]="${serving}-autoscaler-hpa"
  ["controller"]="${serving}-controller"
  ["webhook"]="${serving}-webhook"
  ["storage-version-migration-serving-0.17.3__migrate"]="${serving}-storage-version-migration"

  ["3scale-kourier-gateway"]="docker.io/maistra/proxyv2-ubi8:1.1.0"
  ["3scale-kourier-control"]="${registry}/knative-v0.16.0:kourier"

  ["KN_CLI_ARTIFACTS"]="${registry}/knative-v0.17.2:kn-cli-artifacts"
)

declare -A eventing_images
eventing_images=(
  ["eventing-controller__eventing-controller"]="${eventing}-controller"
  ["sugar-controller__controller"]="${eventing}-sugar-controller"
  ["eventing-webhook__eventing-webhook"]="${eventing}-webhook"
  ["storage-version-migration-eventing__migrate"]="${eventing}-storage-version-migration"

  ["mt-broker-controller__mt-broker-controller"]="${eventing}-mtchannel-broker"
  ["mt-broker-filter__filter"]="${eventing}-mtbroker-filter"
  ["mt-broker-ingress__ingress"]="${eventing}-mtbroker-ingress"
  ["imc-controller__controller"]="${eventing}-channel-controller"
  ["imc-dispatcher__dispatcher"]="${eventing}-channel-dispatcher"

  ["v0.17.0-pingsource-cleanup__pingsource"]="${eventing}-pingsource-cleanup"
  ["PING_IMAGE"]="${eventing}-ping"
  ["MT_PING_IMAGE"]="${eventing}-mtping"
  ["APISERVER_RA_IMAGE"]="${eventing}-apiserver-receive-adapter"
  ["BROKER_INGRESS_IMAGE"]="${eventing}-broker-ingress"
  ["BROKER_FILTER_IMAGE"]="${eventing}-broker-filter"
  ["DISPATCHER_IMAGE"]="${eventing}-channel-dispatcher"
)

declare -A kafka_images
kafka_images=(
  ["kafka-controller-manager__manager"]="${eventing_contrib}-kafka-source-controller"
  ["KAFKA_RA_IMAGE"]="${eventing_contrib}-kafka-source-adapter"

  ["kafka-ch-controller__controller"]="${eventing_contrib}-kafka-channel-controller"
  ["DISPATCHER_IMAGE"]="${eventing_contrib}-kafka-channel-dispatcher"
  ["kafka-ch-dispatcher__dispatcher"]="${eventing_contrib}-kafka-channel-dispatcher"
  ["kafka-webhook__kafka-webhook"]="${eventing_contrib}-kafka-channel-webhook"
)

function add_related_image {
  cat << EOF | yq w -i -s - "$1"
- command: update 
  path: spec.relatedImages[+]
  value:
    name: "IMAGE_${2}_${3}"
    image: "${4}"
EOF
}

function add_downstream_operator_deployment_image {
  cat << EOF | yq w -i -s - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers[0].env[+]
  value:
    name: "IMAGE_${2}_${3}"
    value: "${4}"
EOF
}

# since we also parse the environment variables in the upstream (actually midstream) operator,
# we don't add scope prefixes to image overrides here. We don't have a clash anyway without any scope prefixes!
# there was a naming clash between eventing and kafka, but we won't provide the Kafka overrides to the
# midstream operator.
function add_upstream_operator_deployment_image {
  cat << EOF | yq w -i -s - "$1"
- command: update
  path: spec.install.spec.deployments(name==knative-operator).spec.template.spec.containers[0].env[+]
  value:
    name: "IMAGE_${2}"
    value: "${3}"
EOF
}

# Start fresh
cp "$template" "$target"

for name in "${!serving_images[@]}"; do
  add_related_image "$target" "SERVING" "$name" "${serving_images[$name]}"
  add_downstream_operator_deployment_image "$target" "SERVING" "$name" "${serving_images[$name]}"
  add_upstream_operator_deployment_image "$target" "$name" "${serving_images[$name]}"
done

for name in "${!eventing_images[@]}"; do
  add_related_image "$target" "EVENTING" "$name" "${eventing_images[$name]}"
  add_downstream_operator_deployment_image "$target" "EVENTING" "$name" "${eventing_images[$name]}"
  add_upstream_operator_deployment_image "$target" "$name" "${eventing_images[$name]}"
done

# don't add Kafka image overrides to upstream operator
for name in "${!kafka_images[@]}"; do
  add_related_image "$target" "KAFKA" "$name" "${kafka_images[$name]}"
  add_downstream_operator_deployment_image "$target" "KAFKA" "$name" "${kafka_images[$name]}"
done
