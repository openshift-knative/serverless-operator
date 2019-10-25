#!/usr/bin/env bash

function create_namespaces {
  logger.info 'Create namespaces'
  for ns in "${SERVICE_MESH_MEMBERS[@]}"; do
    if ! oc get ns "${ns}" >/dev/null 2>&1; then
      oc create ns "${ns}"
    fi
  done
  logger.success "Namespaces has bean created: ${SERVICE_MESH_MEMBERS[*]}"
}

function delete_namespaces {
  teardown_service_mesh_member_roll
  logger.info "Deleting namespaces"
  for ns in "${SERVICE_MESH_MEMBERS[@]}"; do
    if oc get ns "${ns}" >/dev/null 2>&1; then
      logger.info "Waiting until there are no pods in ${ns} to safely remove it..."
      timeout 600 "[[ \$(oc get pods -n $ns -o jsonpath='{.items}') != '[]' ]]"
      oc delete ns "$ns"
    fi
  done
  logger.success "Namespaces has been deleted."
}
