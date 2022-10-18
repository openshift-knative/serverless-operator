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
  if [[ $# -eq 0 ]]; then
    echo "Pass an array with namespaces as arg[1]" && exit 1
  fi
  local namespaces
  namespaces=("$@")
  for ns in "${namespaces[@]}"; do
    ensure_namespace "${ns}"
  done
  # Create an OperatorGroup if there are no other ones in the namespace.
  if [[ $(oc get operatorgroups -oname -n "${OPERATORS_NAMESPACE}" | wc -l) -eq 0 ]]; then
    cat <<EOF | oc apply -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: serverless
  namespace: ${OPERATORS_NAMESPACE}
EOF
  fi
  logger.success "Namespaces have been created: ${namespaces[*]}"
}

# Link global pull secrets for accessing private registries, see https://issues.redhat.com/browse/SRVKS-833
# Allows pulling images from a secured registry, e.g. internal mirror registry for disconnected env.
function link_global_pullsecret_to_namespaces {
  logger.info 'Link global pull secret to namespaces'
  if [[ $# -eq 0 ]]; then
    echo "Pass an array with namespaces as arg[1]" && exit 1
  fi
  local namespaces
  namespaces=("$@")
  for ns in "${namespaces[@]}"; do
    if ! oc -n "${ns}" get secret pull-secret &>/dev/null; then
      oc -n openshift-config get secret pull-secret -o yaml | sed "s/namespace: .*/namespace: ${ns}/" | oc apply -f -
    fi
    oc -n "$ns" secrets link default pull-secret --for=pull
  done
}

function delete_namespaces {
  logger.info "Deleting namespaces"
  if [[ $# -eq 0 ]]; then
      echo "Pass an array with namespaces as arg[1]" && exit 1
  fi
  local namespaces
  namespaces=("$@")
  for ns in "${namespaces[@]}"; do
    if oc get ns "${ns}" >/dev/null 2>&1; then
      logger.info "Waiting until there are no pods in ${ns} to safely remove it..."
      timeout 600 "[[ \$(oc get pods -n $ns --field-selector=status.phase!=Succeeded -o jsonpath='{.items}') != '[]' ]]"
      oc delete ns "$ns"
    fi
  done
  logger.success "Namespaces have been deleted: ${namespaces[*]}"
}
