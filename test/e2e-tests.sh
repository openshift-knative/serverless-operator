#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then
  set -x
  env
fi

register_teardown || exit $?
scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_catalogsource || failed=3
(( !failed )) && logger.success 'Cluster prepared for testing.'

# Run serverless-operator specific tests.
(( !failed )) && run_e2e_tests || failed=4

# Run upstream knative serving operator tests
(( !failed )) && deploy_serverless_operator_latest || failed=11
(( !failed )) && run_knative_serving_operator_tests "$KNATIVE_VERSION" || failed=12

# Run upstream knative serving tests
(( !failed )) && ensure_serverless_installed || failed=6
(( !failed )) && run_knative_serving_e2e_and_conformance_tests "$KNATIVE_VERSION" || failed=7
(( !failed )) && teardown_serverless || failed=8

(( failed )) && dump_state
(( failed )) && exit $failed

success
