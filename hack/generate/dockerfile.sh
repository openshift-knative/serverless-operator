#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target Dockerfile file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

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
values[OCP_MAX_VERSION]="$(metadata.get 'requirements.ocpVersion.max')"
values[PREVIOUS_VERSION]="$(metadata.get olm.replaces)"

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s|__${before}__|${values[${before}]}|" "$target"
done

# For index image, append older bundles for the "render" command.
if [[ "$template" =~ index.Dockerfile ]]; then
  current_version=$(metadata.get 'project.version')
  major=$(versions.major "$current_version")
  minor=$(versions.minor "$current_version")
  micro=$(versions.micro "$current_version")

  # One is already added in template
  num_csvs=$(( INDEX_IMAGE_NUM_CSVS-1 ))

  # Generate additional entries
  for i in $(seq $num_csvs); do
    current_minor=$(( minor-$i ))
    # If the current version is a z-stream then the following entries will
    # start with the same "minor" version.
    if [[ "$micro" != "0" ]]; then
      current_minor=$(( current_minor+1 ))
    fi
    current_version="${major}.${current_minor}.0"

    sed --in-place "/opm render/a registry.ci.openshift.org/knative/release-${current_version}:serverless-bundle \\\\" "$target"
  done

  # Hacks. Should gradually go away with next versions.
  # Workaround for https://issues.redhat.com/browse/SRVCOM-3207
  # Use a manually built image for 1.32.0.
  # TODO: Remove this when 1.32.0 is not included in index. This is a problem only for 1.32.0.
  sed --in-place "s|registry.ci.openshift.org/knative/release-1.32.0:serverless-bundle|quay.io/openshift-knative/serverless-bundle:release-1.32.0|" "$target"
  # Replace the old format for 1.31.0 and older.
  sed --in-place "s|registry.ci.openshift.org/knative/release-1.31.0:serverless-bundle|registry.ci.openshift.org/knative/openshift-serverless-v1.31.0:serverless-bundle|" "$target"
fi

