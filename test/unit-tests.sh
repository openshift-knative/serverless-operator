#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

debugging.setup

gotestsum=gotest.tools/gotestsum@v1.8.2
declare -A testsuites=(
  [knative-operator]=knative-operator
  [openshift-knative-operator]=openshift-knative-operator
  [serving-ingress]=serving/ingress
  [serving-metadata-webhook]=serving/metadata-webhook
)

IMAGE_TEMPLATE=''

for ts in "${!testsuites[@]}"; do
  logger.info "Testing $ts"
  go run "$gotestsum" \
	  --format testname \
	  --junitfile "${ARTIFACTS:-/tmp}/junit_${ts}.xml" \
	  -- -count=1 -race "./${testsuites[$ts]}/..."
done

logger.success 'Unit tests passed'
