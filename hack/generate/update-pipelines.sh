#!/usr/bin/env bash

set -euo pipefail

readonly version=$(yq read olm-catalog/serverless-operator/project.yaml project.version)
# shellcheck disable=SC2038
find .tekton/ \( -name "*-push.yaml" \) | xargs -I{} yq write --inplace "{}" "spec.params(name==additional-tags).value[0]" "${version}"
