#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

initialize || exit $?
scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_service_mesh_operator || failed=2
(( !failed )) && install_catalogsource || failed=3
(( !failed )) && logger.success 'Cluster prepared for testing.'

# Run serverless-operator specific tests.
(( !failed )) && run_e2e_tests || failed=4

# Setup serverless and run upstream e2e and conformance tests.
(( !failed )) && ensure_serverless_installed || failed=5
(( !failed )) && run_upstream_tests "v0.9.0" || failed=6
(( !failed )) && teardown_serverless || failed=7

(( failed )) && dump_state
(( failed )) && exit $failed

success
