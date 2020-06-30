#!/usr/bin/env bash

function create_namespaces {
  logger.info 'Create namespaces'
  for ns in "${NAMESPACES[@]}"; do
    if ! oc get ns "${ns}" >/dev/null 2>&1; then
      oc create ns "${ns}"
    fi
  done
  cat <<EOF | oc apply -f - || return $?
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: serverless
  namespace: ${OPERATORS_NAMESPACE}
EOF
  logger.success "Namespaces has bean created: ${NAMESPACES[*]}"
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
  logger.success "Namespaces has been deleted."
}
