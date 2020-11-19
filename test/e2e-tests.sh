#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  env
fi
debugging.setup

scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_catalogsource || failed=1
(( !failed )) && logger.success '🚀 Cluster prepared for testing.'

# Run serverless-operator specific tests.
(( !failed )) && serverless_operator_e2e_tests || failed=2
if [[ $TEST_KNATIVE_KAFKA == true ]]; then
  (( !failed )) && install_strimzi || failed=3
  (( !failed )) && serverless_operator_kafka_e2e_tests || failed=4
fi

(( !failed )) && ensure_serverless_installed || failed=5
# Run Knative Serving & Eventing downstream E2E tests.
(( !failed )) && downstream_serving_e2e_tests || failed=6
(( !failed )) && downstream_eventing_e2e_tests || failed=7

if [[ $TEST_KNATIVE_KAFKA == true ]]; then
 (( !failed )) && ensure_kafka_no_auth || failed=8
 (( !failed )) && downstream_knative_kafka_e2e_tests || failed=9
# (( !failed )) && ensure_kafka_tls_auth || failed=10
# (( !failed )) && downstream_knative_kafka_e2e_tests || failed=11
# (( !failed )) && ensure_kafka_sasl_auth || failed=12
# (( !failed )) && downstream_knative_kafka_e2e_tests || failed=13
fi

(( failed )) && dump_state
(( failed )) && exit $failed

success
