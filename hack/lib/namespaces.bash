#!/usr/bin/env bash

function ensure_namespace {
  local ns
  ns="${1:?Pass namespace name as arg[1]}"
  if ! oc get namespace "${ns}" >/dev/null 2>&1; then
    oc create namespace "${ns}"
  fi
}

function create_namespaces {
  logger.info 'Create namespaces'
  for ns in "${NAMESPACES[@]}"; do
    ensure_namespace "${ns}"
  done
  # Create an OperatorGroup if there are no other ones in the namespace.
  if [[ $(oc get operatorgroups -oname -n "${OPERATORS_NAMESPACE}" | wc -l) -eq 0 ]]; then
    cat <<EOF | oc apply -f - || return $?
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: serverless
  namespace: ${OPERATORS_NAMESPACE}
EOF
  fi
  logger.success "Namespaces have been created: ${NAMESPACES[*]}"
}

function delete_namespaces {
  logger.info "Deleting namespaces"
  for ns in "${NAMESPACES[@]}"; do
    if oc get ns "${ns}" >/dev/null 2>&1; then
      logger.info "Waiting until there are no pods in ${ns} to safely remove it..."
      timeout 600 "[[ \$(oc get pods -n $ns --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
      oc delete ns "$ns"
    fi
  done
  logger.success "Namespaces have been deleted."
}
