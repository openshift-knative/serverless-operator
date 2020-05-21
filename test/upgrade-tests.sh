#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib.bash"

set -Eeuo pipefail

# Enable extra verbosity if running in CI.
if [ -n "$OPENSHIFT_BUILD_NAMESPACE" ]; then
  env
fi
debugging.setup

scale_up_workers || exit $?
create_namespaces || exit $?
create_htpasswd_users && add_roles || exit $?

failed=0

(( !failed )) && install_catalogsource || failed=3
(( !failed )) && logger.success 'Cluster prepared for testing.'

# Test upgrades from CSV one version back, with manual approval.
(( !failed )) && install_serverless_previous || failed=4
(( !failed )) && run_knative_serving_rolling_upgrade_tests || failed=5
(( !failed )) && teardown_serverless || failed=6

# Test upgrades from CSV two versions back to the latest, with automatic approval.
upgrade_from="$(latest_minus_two_csv)" || return $?
(( !failed )) && INITIAL_CSV="$upgrade_from" install_serverless_previous || failed=7
(( !failed )) && INITIAL_CSV="$upgrade_from" AUTO_UPGRADES=true run_knative_serving_rolling_upgrade_tests || failed=8
(( !failed )) && teardown_serverless || failed=9

# Test upgrades from oldest compatible CSV to the latest, with automatic approval.
upgrade_from="$(oldest_compatible_csv "$OLM_UPGRADE_CHANNEL")" || return $?
(( !failed )) && INITIAL_CSV="$upgrade_from" install_serverless_previous || failed=10
(( !failed )) && INITIAL_CSV="$upgrade_from" AUTO_UPGRADES=true run_knative_serving_rolling_upgrade_tests || failed=11

echo ">>> Knative Servings"
oc get knativeserving.operator.knative.dev --all-namespaces -o yaml

echo ">>> Knative Services"
oc get ksvc --all-namespaces

echo ">>> Triggering GC"
for pod in $(oc get pod -n openshift-kube-controller-manager -l kube-controller-manager=true -o custom-columns=name:metadata.name --no-headers); do
  echo "killing pod $pod"
  oc rsh -n openshift-kube-controller-manager "$pod" /bin/sh -c "kill 1"
  sleep 30
done

echo "Sleeping so GC can run"
sleep 120

echo ">>> Knative Servings"
oc get knativeserving.operator.knative.dev --all-namespaces -o yaml

echo ">>> Knative Services"
oc get ksvc --all-namespaces

(( failed )) && dump_state
(( failed )) && exit $failed

success
