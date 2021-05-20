#!/usr/bin/env bash

set -Eeuo pipefail

root="$(dirname "${BASH_SOURCE[0]}")/../../.."

# Source the main vars file to get the operator version to be used.
# shellcheck disable=SC1091,SC1090
source "$root/hack/lib/__sources__.bash"

version=${OPERATOR_VERSION:-v$(metadata.get '.dependencies.operator')}

target_dir="$root/olm-catalog/serverless-operator/manifests/"
target_serving_file="$target_dir/operator_v1alpha1_knativeserving_crd.yaml"
target_eventing_file="$target_dir/operator_v1alpha1_knativeeventing_crd.yaml"
rm -rf "$target_serving_file" "$target_eventing_file"

serving_url="https://raw.githubusercontent.com/knative/operator/$version/config/300-serving.yaml"
eventing_url="https://raw.githubusercontent.com/knative/operator/$version/config/300-eventing.yaml"

wget --no-check-certificate "$serving_url" -O "$target_serving_file"
wget --no-check-certificate "$eventing_url" -O "$target_eventing_file"

# For SRVKE-755 state the actual default for Openshift Serverless disable HPA:
git apply "$root/olm-catalog/serverless-operator/hack/001-eventing-sinkbinding-default-override"

# Drop unsupported fields from the Serving CRD.
git apply "$root/olm-catalog/serverless-operator/hack/002-serving-drop-unsupported-fields.patch"

# Drop unsupported fields from the Eventing CRD.
git apply "$root/olm-catalog/serverless-operator/hack/003-eventing-drop-unsupported-fields.patch"
