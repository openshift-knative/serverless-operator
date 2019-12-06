#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

register_teardown || exit $?
scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_service_mesh_operator || failed=2
(( !failed )) && install_catalogsource || failed=3
(( !failed )) && logger.success 'Cluster prepared for testing.'

(( !failed )) && install_serverless_previous || failed=5
(( !failed )) && run_knative_serving_rolling_upgrade_tests "v0.10.0" || failed=6

(( !failed )) && teardown_serverless || failed=7

(( failed )) && dump_state
(( failed )) && exit $failed

success
