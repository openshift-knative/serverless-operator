#!/usr/bin/env bash

set -Eeuo pipefail

target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_channel_entries {
  local channel_entry_yaml channel target current_version replaces_version base_version
  channel=${1:?Provide channel name}
  target=${2:?Provide target file name}
  channel_entry_yaml="$(mktemp -t default-entry-XXXXX.yaml)"

  # Initialize a temporary file for a single entry.
  cat > "${channel_entry_yaml}" <<EOF
schema: olm.channel
name: ${channel}
package: serverless-operator
entries:
EOF

  current_version=$(metadata.get 'project.version')
  major=$(versions.major "$current_version")
  minor=$(versions.minor "$current_version")
  micro=$(versions.micro "$current_version")

  for i in $(seq "$INDEX_IMAGE_NUM_CSVS"); do
    current_major=$((major))
    current_minor=$((minor))
    current_micro=$((micro - i + 1))

    if [[ "$current_micro" -le 0 ]]; then
      current_minor=$((minor + current_micro))
      current_micro=0
    fi

    current_version="${current_major}.${current_minor}.${current_micro}"
    if [[ "$current_micro" == "0" ]]; then
      base_version="${current_major}.$(( current_minor-1 )).0"
      replaces_version="${current_major}.$(( current_minor-1 )).0"
    else
      base_version="${current_major}.${current_minor}.0"
      replaces_version="${current_major}.${current_minor}.$(( current_micro-1 ))"
    fi

    if [[ $i -eq "$INDEX_IMAGE_NUM_CSVS" ]]; then
      yq write --inplace "$channel_entry_yaml" 'entries[+].name' "serverless-operator.v${current_version}"
    else
      cat << EOF | yq write --inplace --script - "$channel_entry_yaml"
  - command: update
    path: entries[+]
    value:
      name: "serverless-operator.v${current_version}"
      replaces: "serverless-operator.v${replaces_version}"
      skipRange: ">=${base_version} <${current_version}"
EOF
    fi
  done

  echo "---" >> "${target}"
  cat "${channel_entry_yaml}" >> "${target}"
}

# Clear the file.
rm -f "${target}"

while IFS=$'\n' read -r channel; do
  add_channel_entries "$channel" "${target}"
done < <(metadata.get 'olm.channels.list[*]')
