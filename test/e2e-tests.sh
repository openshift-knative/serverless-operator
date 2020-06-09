#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then
  env
fi
debugging.setup

failed=1
rm -f "${KUBECONFIG}" # Give must-gather signal to not run

(( failed )) && exit $failed

success
