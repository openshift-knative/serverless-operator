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

(( !failed )) && install_catalogsource || failed=3
(( !failed )) && logger.success 'Cluster prepared for testing.'

(( !failed )) && install_serverless_previous || failed=5
(( !failed )) && run_knative_serving_rolling_upgrade_tests || failed=6

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
