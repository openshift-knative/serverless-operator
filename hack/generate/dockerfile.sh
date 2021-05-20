#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target Dockerfile file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

declare -A values
values[NAME]="$(metadata.get .project.name)"
values[CHANNEL_LIST]="$(metadata.get '.olm.channels.list[]' | paste -sd ',' -)"
values[DEFAULT_CHANNEL]="$(metadata.get .olm.channels.default)"
values[VERSION]="$(metadata.get .project.version)"
values[SERVING_VERSION]="$(metadata.get .dependencies.serving)"
values[EVENTING_VERSION]="$(metadata.get .dependencies.eventing)"
values[EVENTING_KAFKA_VERSION]="$(metadata.get .dependencies.eventing_kafka)"
values[GOLANG_VERSION]="$(metadata.get .requirements.golang)"
values[NODEJS_VERSION]="$(metadata.get .requirements.nodejs)"
values[OCP_TARGET_VLIST]="$(metadata.get '.requirements.ocp[]' | sed 's/^/v/' | paste -sd ',' -)"

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s/__${before}__/${values[${before}]}/" "$target"
done
