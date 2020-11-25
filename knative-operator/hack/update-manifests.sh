#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../.."

# Source the main vars file to get the serving/eventing version to be used.
source "$root/hack/lib/__sources__.bash"

version=${KOURIER_VERSION:-v$(metadata.get dependencies.kourier)}

target_dir="$root/knative-operator/deploy/resources/kourier"
rm -rf "$target_dir"
mkdir -p "$target_dir"

target_file="$target_dir/kourier-latest.yaml"

url="https://github.com/knative-sandbox/net-kourier/releases/download/$version/kourier.yaml"
wget --no-check-certificate "$url" -O "$target_file"

# TODO: [SRVKS-610] These values should be replaced by operator instead of sed.
sed -i -e 's/kourier-control.knative-serving/kourier-control.knative-serving-ingress/g' "$target_file"
