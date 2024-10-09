#!/usr/bin/env bash

set -Eeuo pipefail

target="${1:?Provide a target index yaml file as arg[1]}"

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../lib/__sources__.bash"

function add_channel_entries {
  local channel_entry_yaml channel target num_csvs
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

  # Handle the first entry specifically as it might be a z-stream release.
  if [[ "$micro" == "0" ]]; then
    previous_version="${major}.$(( minor-1 )).${micro}"
  else
    previous_version="${major}.${minor}.0"
  fi

  cat << EOF | yq write --inplace --script - "$channel_entry_yaml"
  - command: update
    path: entries[+]
    value:
      name: "serverless-operator.v${current_version}"
      replaces: "serverless-operator.v${previous_version}"
      skipRange: ">=${previous_version} <${current_version}"
EOF

  # One is already added above specifically
  num_csvs=$(( INDEX_IMAGE_NUM_CSVS-1 ))

  # Generate additional entries
  for i in $(seq $num_csvs); do
    current_minor=$(( minor-i ))
    previous_minor=$(( minor-i ))
    previous_minor=$(( previous_minor-1 ))
    # If the current version is a z-stream then the following entries will
    # start with the same "minor" version.
    if [[ "$micro" != "0" ]]; then
      current_minor=$(( current_minor+1 ))
      previous_minor=$(( previous_minor+1 ))
    fi

    current_version="${major}.${current_minor}.0"
    previous_version="${major}.${previous_minor}.0"

    # If this is the last item enter only name, without "replaces".
    if [[ $i -eq $num_csvs ]]; then
      yq write --inplace "$channel_entry_yaml" 'entries[+].name' "serverless-operator.v${current_version}"
    else
      cat << EOF | yq write --inplace --script - "$channel_entry_yaml"
  - command: update
    path: entries[+]
    value:
      name: "serverless-operator.v${current_version}"
      replaces: "serverless-operator.v${previous_version}"
      skipRange: ">=${previous_version} <${current_version}"
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
