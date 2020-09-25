#!/bin/bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

declare -A images
images=(
  ["queue-proxy"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-queue"
  ["activator"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-activator"
  ["autoscaler"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-autoscaler"
  ["autoscaler-hpa"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-autoscaler-hpa"
  ["controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-controller"
  ["webhook"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-webhook"
  ["storage-version-migration-serving-0.16.0__migrate"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-serving-storage-version-migration"

  ["3scale-kourier-gateway"]="docker.io/maistra/proxyv2-ubi8:1.1.0"
  ["3scale-kourier-control"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:kourier"

  ["eventing-controller__eventing-controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-controller"
  ["sugar-controller__controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-sugar-controller"
  ["eventing-webhook__eventing-webhook"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-webhook"
  ["storage-version-migration-eventing__migrate"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-storage-version-migration"

  # the next three images have 0 replicas, they got removed, and point to 0.15 release images to migrate their deployments to 0
  ["broker-controller__broker-controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.15.2:knative-eventing-channel-broker"
  ["broker-filter__filter"]="registry.svc.ci.openshift.org/openshift/knative-v0.15.2:knative-eventing-broker-filter"
  ["broker-ingress__ingress"]="registry.svc.ci.openshift.org/openshift/knative-v0.15.2:knative-eventing-broker-ingress"

  # the mt broker replaces the removed broker
  ["mt-broker-controller__mt-broker-controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-mtchannel-broker"
  ["mt-broker-filter__filter"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-mtbroker-filter"
  ["mt-broker-ingress__ingress"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-mtbroker-ingress"
  ["imc-controller__controller"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-channel-controller"
  ["imc-dispatcher__dispatcher"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-channel-dispatcher"

  ["v0.16.0-broker-cleanup__brokers"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-broker-cleanup"
  ["PING_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-ping"
  ["MT_PING_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-mtping"
  ["APISERVER_RA_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-apiserver-receive-adapter"
  ["BROKER_INGRESS_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-broker-ingress"
  ["BROKER_FILTER_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-broker-filter"
  ["DISPATCHER_IMAGE"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.0:knative-eventing-channel-dispatcher"

  ["KN_CLI_ARTIFACTS"]="registry.svc.ci.openshift.org/openshift/knative-v0.16.1:kn-cli-artifacts"
)

function add_image {
  cat << EOF | yq w -i -s - "$1"
- command: update 
  path: spec.relatedImages[+]
  value:
    name: "IMAGE_${2}"
    image: "${3}"
EOF

  cat << EOF | yq w -i -s - "$1"
- command: update 
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers[0].env[+]
  value:
    name: "IMAGE_${2}"
    value: "${3}"
EOF
}

# Start fresh
cp "$template" "$target"

for name in "${!images[@]}"; do
  add_image "$target" "$name" "${images[$name]}"
done