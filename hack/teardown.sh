#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

set -Eeuo pipefail

declare -a TEST_KN_NAMESPACE
TEST_KN_NAMESPACE=("serverless-tests" "serverless-tests1" "serverless-tests2" "serverless-tests3" "serverless-tests-mesh")
#export TEST_KN_NAMESPACE

function teardown_test_data {
   echo "Teardown operator test data 💀"
   for NS in "${TEST_KN_NAMESPACE[@]}"; do
      if oc get ns "${NS}" >/dev/null 2>&1; then
         echo "Removing resources in $NS namespace"
         oc delete --ignore-not-found=true --all -n "$NS" deployment >/dev/null 2>&1
         oc delete --ignore-not-found=true --all -n "$NS" deployment.apps >/dev/null 2>&1
         oc delete --ignore-not-found=true --all -n "$NS" replicaset.apps >/dev/null 2>&1
         oc delete --ignore-not-found=true --all -n "$NS" pods >/dev/null 2>&1
         oc delete --ignore-not-found=true --all -n "$NS" service >/dev/null 2>&1
         oc delete --ignore-not-found=true --all -n "$NS" imagestream.image.openshift.io >/dev/null 2>&1
         if oc get knativeservings.operator.knative.dev -n knative-serving >/dev/null 2>&1; then
            oc delete --ignore-not-found=true --all -n "$NS" revision.serving.knative.dev >/dev/null 2>&1
            oc delete --ignore-not-found=true --all -n "$NS" route.serving.knative.dev >/dev/null 2>&1
            oc delete --ignore-not-found=true --all -n "$NS" service.serving.knative.dev >/dev/null 2>&1
            oc delete --ignore-not-found=true --all -n "$NS" configuration.serving.knative.dev >/dev/null 2>&1
         fi
         oc delete ns "$NS" >/dev/null 2>&1
      fi
   done
   echo "Teardown operator test data completed 🌟"
}

debugging.setup

teardown_test_data
teardown_serverless
teardown_extras
teardown_tracing
uninstall_mesh
delete_catalog_source
delete_namespaces
delete_namespaces "${SYSTEM_NAMESPACES[@]}"

# display data
sleep 30
oc get all -n knative-serving
oc get all -n knative-serving-ingress
oc get all -n knative-eventing
oc get all -n openshift-operators
oc get all -n openshift-serverless
oc get all -n istio-system
oc get all -n serverless
for NS in "${TEST_KN_NAMESPACE[@]}"; do
   oc get all -n "$NS"
done
