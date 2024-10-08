#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target Dockerfile file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

default_serverless_operator_images

declare -A values
values[NAME]="$(metadata.get project.name)"
values[LATEST_VERSIONED_CHANNEL]="$(metadata.get 'olm.channels.list[1]')"
values[DEFAULT_CHANNEL]="$(metadata.get olm.channels.default)"
values[VERSION]="$(metadata.get project.version)"
values[SERVING_VERSION]="$(metadata.get dependencies.serving)"
values[EVENTING_VERSION]="$(metadata.get dependencies.eventing)"
values[EVENTING_KAFKA_BROKER_VERSION]="$(metadata.get dependencies.eventing_kafka_broker)"
values[EVENTING_ISTIO_VERSION]="$(metadata.get dependencies.eventing_istio)"
values[GOLANG_VERSION]="$(metadata.get requirements.golang)"
values[NODEJS_VERSION]="$(metadata.get requirements.nodejs)"
values[OCP_TARGET_VLIST]="$(metadata.get 'requirements.ocpVersion.label')"
values[OCP_MAX_VERSION]="$(metadata.get 'requirements.ocpVersion.list[-1]')"
values[PREVIOUS_VERSION]="$(metadata.get olm.replaces)"
values[BUNDLE]="${SERVERLESS_BUNDLE}"

# For index image, append older bundles for the "render" command.
if [[ "$template" =~ index.Dockerfile ]]; then
  while IFS=$'\n' read -r ocp_version; do
    values[OCP_VERSION]="${ocp_version}"
    target_dockerfile="${target}/v${ocp_version}/Dockerfile"

    cp "$template" "${target_dockerfile}"

    for before in "${!values[@]}"; do
      echo "Value: ${before} -> ${values[$before]}"
      sed --in-place "s|__${before}__|${values[${before}]}|" "${target_dockerfile}"
    done
  done < <(metadata.get 'requirements.ocpVersion.list[*]')
else
  cp "$template" "$target"

  for before in "${!values[@]}"; do
    echo "Value: ${before} -> ${values[$before]}"
    sed --in-place "s|__${before}__|${values[${before}]}|" "$target"
  done
fi

