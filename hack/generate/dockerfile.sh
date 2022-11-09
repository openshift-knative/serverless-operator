#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target Dockerfile file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

declare -A values
values[NAME]="$(metadata.get project.name)"
values[CHANNEL_LIST]="$(metadata.get 'olm.channels.list.*' | paste -sd ',' -)"
values[DEFAULT_CHANNEL]="$(metadata.get olm.channels.default)"
values[VERSION]="$(metadata.get project.version)"
values[VERSION_MAJOR_MINOR]="$(cut -d '.' -f 1 <<< "${values[VERSION]}")"."$(cut -d '.' -f 2 <<< "${values[VERSION]}")"
values[SERVING_VERSION]="$(metadata.get dependencies.serving)"
values[EVENTING_VERSION]="$(metadata.get dependencies.eventing)"
values[EVENTING_KAFKA_VERSION]="$(metadata.get dependencies.eventing_kafka)"
values[EVENTING_KAFKA_BROKER_VERSION]="$(metadata.get dependencies.eventing_kafka_broker)"
values[GOLANG_VERSION]="$(metadata.get requirements.golang)"
values[NODEJS_VERSION]="$(metadata.get requirements.nodejs)"
values[OCP_TARGET_VLIST]="$(metadata.get 'requirements.ocpVersion.label')"
values[PREVIOUS_VERSION]="$(metadata.get olm.replaces)"

# If EKB dependency information has a length smaller than 10, then we assume it is something like 1.2 or 1.2.3 and go with
# using a single integration stream (https://docs.ci.openshift.org/docs/architecture/ci-operator/#publishing-to-an-integration-stream).
#
# Integration streams created by OpenShift CI put the component name as the tag.
# example: registry/knative-v1.2.3:kafka-broker-dispatcher
# example: registry/knative-v1.2.3:kafka-broker-receiver
#
# However, if EKB dependency information is longer than 3, we assume it is a commit hash like deadbeef. In that case
# we go with using one integration stream per component which are tagged by commits.
# (https://docs.ci.openshift.org/docs/architecture/ci-operator/#publishing-images-tagged-by-commit)
#
# In that case, we would like to use the images that aren't labeled with a floating tag.
# example: registry/kafka-broker-dispatcher:deadbeef
# example: registry/kafka-broker-receiver:deadbeef
#
ekb_version=$(metadata.get dependencies.eventing_kafka_broker)
if [ ${#ekb_version} -lt 6 ]; then
  values[EVENTING_KAFKA_SRC_IMAGE]="knative-v${ekb_version}:eventing-kafka-broker-src"
else
  values[EVENTING_KAFKA_SRC_IMAGE]="eventing-kafka-broker-src:${ekb_version}"
fi

# Start fresh
cp "$template" "$target"

for before in "${!values[@]}"; do
  echo "Value: ${before} -> ${values[$before]}"
  sed --in-place "s/__${before}__/${values[${before}]}/" "$target"
done
