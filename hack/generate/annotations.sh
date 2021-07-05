#!/usr/bin/env bash

set -Eeuo pipefail

template="${1:?Provide template file as arg[1]}"
target="${2:?Provide a target annotations file as arg[2]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/metadata.bash"

declare -A values
values[operators.operatorframework.io.bundle.channel.default.v1]="$(metadata.get .olm.channels.default)"
values[operators.operatorframework.io.bundle.channels.v1]="$(metadata.get '.olm.channels.list[]' | paste -sd ',' -)"
values[operators.operatorframework.io.bundle.package.v1]="$(metadata.get .project.name)"

# Start fresh
cp "$template" "$target"

for name in "${!values[@]}"; do
  echo "Value: ${name} -> ${values[$name]}"
  yq e --inplace ".annotations[\"$name\"] = \"${values[$name]}\"" "$target"
done
