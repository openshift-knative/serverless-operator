#!/bin/bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target CSV file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

registry="registry.svc.ci.openshift.org/openshift"
serving="${registry}/knative-v$(metadata.get dependencies.serving):knative-serving"
eventing="${registry}/knative-v$(metadata.get dependencies.eventing):knative-eventing"

declare -A images
images["queue-proxy"]="${serving}-queue"
images["activator"]="${serving}-activator"
images["autoscaler"]="${serving}-autoscaler"
images["autoscaler-hpa"]="${serving}-autoscaler-hpa"
images["controller"]="${serving}-controller"
images["webhook"]="${serving}-webhook"
images["storage-version-migration-serving-$(metadata.get dependencies.serving)__migrate"]="${serving}-storage-version-migration"

images["3scale-kourier-gateway"]="docker.io/maistra/proxyv2-ubi8:$(metadata.get dependencies.maistra)"
images["3scale-kourier-control"]="${registry}/knative-v$(metadata.get dependencies.kourier):kourier"

images["eventing-controller__eventing-controller"]="${eventing}-controller"
images["sugar-controller__controller"]="${eventing}-sugar-controller"
images["eventing-webhook__eventing-webhook"]="${eventing}-webhook"
images["storage-version-migration-eventing__migrate"]="${eventing}-storage-version-migration"

images["mt-broker-controller__mt-broker-controller"]="${eventing}-mtchannel-broker"
images["mt-broker-filter__filter"]="${eventing}-mtbroker-filter"
images["mt-broker-ingress__ingress"]="${eventing}-mtbroker-ingress"
images["imc-controller__controller"]="${eventing}-channel-controller"
images["imc-dispatcher__dispatcher"]="${eventing}-channel-dispatcher"

images["v$(metadata.get dependencies.knative-release)-pingsource-cleanup__pingsource"]="${eventing}-pingsource-cleanup"
images["PING_IMAGE"]="${eventing}-ping"
images["MT_PING_IMAGE"]="${eventing}-mtping"
images["APISERVER_RA_IMAGE"]="${eventing}-apiserver-receive-adapter"
images["BROKER_INGRESS_IMAGE"]="${eventing}-broker-ingress"
images["BROKER_FILTER_IMAGE"]="${eventing}-broker-filter"
images["DISPATCHER_IMAGE"]="${eventing}-channel-dispatcher"

images["KN_CLI_ARTIFACTS"]="${registry}/knative-v$(metadata.get dependencies.cli):kn-cli-artifacts"

declare -A values
values[spec.version]="$(metadata.get project.version)"
values[metadata.name]="$(metadata.get project.name).v$(metadata.get project.version)"
values['metadata.annotations[olm.skipRange]']="$(metadata.get olm.skipRange)"
values[spec.minKubeVersion]="$(metadata.get requirements.kube.minVersion)"
values[spec.replaces]="$(metadata.get project.name).v$(metadata.get olm.replaces)"

function add_image {
  cat << EOF | yq write --inplace --script - "$1"
- command: update 
  path: spec.relatedImages[+]
  value:
    name: "IMAGE_${2}"
    image: "${3}"
EOF

  cat << EOF | yq write --inplace --script - "$1"
- command: update 
  path: spec.install.spec.deployments(name==knative-openshift).spec.template.spec.containers[0].env[+]
  value:
    name: IMAGE_${2}
    value: ${3}
EOF

  cat << EOF | yq write --inplace --script - "$1"
- command: update 
  path: spec.install.spec.deployments(name==knative-operator).spec.template.spec.containers[0].env[+]
  value:
    name: IMAGE_${2}
    value: ${3}
EOF
}

# Start fresh
cp "$template" "$target"

# Sort images to always produce the same output
keys=()
while IFS='' read -r line; do keys+=("$line"); done < <(echo "${!images[@]}" | tr ' ' $'\n' | sort)

for name in "${keys[@]}"; do
  echo "Image: ${name} -> ${images[$name]}"
  add_image "$target" "$name" "${images[$name]}"
done

for name in "${!values[@]}"; do
  echo "Value: ${name} -> ${values[$name]}"
  yq write --inplace "$target" "$name" "${values[$name]}"
done
