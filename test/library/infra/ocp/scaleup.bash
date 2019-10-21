#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash

function scale_up_workers {
  local cluster_api_ns="openshift-machine-api"
  logger.info 'Scaling cluster up'
  if [[ "${SCALE_UP}" != "true" ]]; then
    logger.info 'Skipping scaling up, because SCALE_UP is set to true.'
    return 0
  fi

  logger.debug 'Get the name of the first machineset that has at least 1 replica'
  local machineset
  machineset=$(oc get machineset -n ${cluster_api_ns} -o custom-columns="name:{.metadata.name},replicas:{.spec.replicas}" | grep -e " [1-9]" | head -n 1 | awk '{print $1}')
  logger.debug "Name found: ${machineset}"

  logger.info 'Bump the number of replicas to 6 (+ 1 + 1 == 8 workers)'
  oc patch machineset -n ${cluster_api_ns} "${machineset}" -p '{"spec":{"replicas":6}}' --type=merge
  wait_until_machineset_scales_up ${cluster_api_ns} "${machineset}" 6
}

# Waits until the machineset in the given namespaces scales up to the
# desired number of replicas
# Parameters: $1 - namespace
#             $2 - machineset name
#             $3 - desired number of replicas
function wait_until_machineset_scales_up() {
  logger.info "Waiting until machineset $2 in namespace $1 scales up to $3 replicas"
  local available
  for _ in {1..150}; do  # timeout after 15 minutes
    available=$(oc get machineset -n "$1" "$2" -o jsonpath="{.status.availableReplicas}")
    if [[ ${available} -eq $3 ]]; then
      echo ''
      logger.info "Machineset $2 successfully scaled up to $3 replicas"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for machineset $2 in namespace $1 to scale up to $3 replicas"
  return 1
}
