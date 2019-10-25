#!/usr/bin/env bash

function scale_up_workers {
  local spec machineset machineset_stat replicas
  logger.info 'Scaling cluster'
  if [[ "${SCALE_UP}" -lt "0" ]]; then
    logger.info 'Skipping scaling up, because SCALE_UP is negative.'
    return 0
  fi

  logger.debug 'Get the machineset with most replicas'
  local machineset
  machineset_stat="$(oc get machineset -n openshift-machine-api -o custom-columns="replicas:{.spec.replicas},name:{.metadata.name}" | tail -n +2 | sort -n | tail -n 1)"
  machineset=$(echo "${machineset_stat}" | awk '{print $2}')
  replicas=$(echo "${machineset_stat}" | awk '{print $1}')
  logger.debug "Name found: ${machineset}, replicas: ${replicas}"

  if [[ "${SCALE_UP}" == "${replicas}" ]]; then
    logger.success "Cluster is already scaled up to ${SCALE_UP} replicas (machine set: ${machineset})."
    return 0
  fi

  logger.info "Bump the number of replicas to ${SCALE_UP}"
  spec="{\"spec\":{\"replicas\": ${SCALE_UP}}}"
  oc patch machineset -n openshift-machine-api "${machineset}" -p "${spec}" --type=merge
  wait_until_machineset_scales_up openshift-machine-api "${machineset}" "${SCALE_UP}"
}

# Waits until the machineset in the given namespaces scales up to the
# desired number of replicas
# Parameters: $1 - namespace
#             $2 - machineset name
#             $3 - desired number of replicas
function wait_until_machineset_scales_up {
  logger.info "Waiting until machineset $2 in namespace $1 scales up to $3 replicas"
  local available
  for _ in {1..150}; do  # timeout after 15 minutes
    available=$(oc get machineset -n "$1" "$2" -o jsonpath="{.status.availableReplicas}")
    if [[ ${available} -eq $3 ]]; then
      echo ''
      logger.success "Machineset $2 successfully scaled up to $3 replicas"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for machineset $2 in namespace $1 to scale up to $3 replicas"
  return 1
}
