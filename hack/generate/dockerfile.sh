#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target Dockerfile file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

declare -A values
values[NAME]="$(metadata.get project.name)"
values[LATEST_VERSIONED_CHANNEL]="$(metadata.get 'olm.channels.list[*]' | head -n 2 | tail -n 1)"
values[DEFAULT_CHANNEL]="$(metadata.get olm.channels.default)"
values[VERSION]="$(metadata.get project.version)"
values[SERVING_VERSION]="$(metadata.get dependencies.serving)"
values[EVENTING_VERSION]="$(metadata.get dependencies.eventing)"
values[EVENTING_KAFKA_BROKER_VERSION]="$(metadata.get dependencies.eventing_kafka_broker)"
values[EVENTING_ISTIO_VERSION]="$(metadata.get dependencies.eventing_istio)"
values[GOLANG_VERSION]="$(metadata.get requirements.golang)"
values[NODEJS_VERSION]="$(metadata.get requirements.nodejs)"
values[OCP_TARGET_VLIST]="$(metadata.get 'requirements.ocpVersion.label')"
values[OCP_MAX_VERSION]="$(metadata.get 'requirements.ocpVersion.max')"
values[PREVIOUS_VERSION]="$(metadata.get olm.replaces)"
values[PREVIOUS_REPLACES]="$(metadata.get olm.previous.replaces)"


prev_prev_channel="$(metadata.get 'olm.channels.list[*]' | head -n 4 | tail -n 1)" # stable-1.32
prev_prev_version="${prev_prev_channel#stable-}.0"

echo "Comparing '${values[PREVIOUS_REPLACES]}' -- '$prev_prev_version'"

if [[ "${values[PREVIOUS_REPLACES]}" != "$prev_prev_version" ]]; then
  values[PREVIOUS_PREVIOUS_VERSION]="registry.ci.openshift.org/knative/openshift-serverless-v${prev_prev_version}:serverless-bundle"
else
  values[PREVIOUS_PREVIOUS_VERSION]=""
fi

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s|__${before}__|${values[${before}]}|" "$target"
done
