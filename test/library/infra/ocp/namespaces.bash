#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash
include infra/await.bash

function create_namespaces {
  logger.info 'Create namespaces'
  oc create ns "$TEST_NAMESPACE"
  oc create ns "$SERVING_NAMESPACE"
}

function delete_namespaces {
  teardown_service_mesh_member_roll
  logger.info "Deleting namespaces"
  timeout 600 "[[ \$(oc get pods -n $TEST_NAMESPACE -o jsonpath='{.items}') != '[]' ]]"
  oc delete namespace "$TEST_NAMESPACE"
  timeout 600 "[[ \$(oc get pods -n $SERVING_NAMESPACE -o jsonpath='{.items}') != '[]' ]]"
  oc delete namespace "$SERVING_NAMESPACE"
}
