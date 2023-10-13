#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup # both install and test
dump_state.setup # test

logger.success '🚀 Cluster prepared for testing.'

run_testselect

# Run serverless-operator kitchensink tests.
downstream_kitchensink_e2e_tests "$@"

success
