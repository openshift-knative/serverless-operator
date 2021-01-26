#!/usr/bin/env bash

function scale_up_workers {
  local current_total az_total replicas idx
  if [[ "${SCALE_UP}" -lt "0" ]]; then
    logger.info 'Skipping scaling up, because SCALE_UP is negative.'
    return 0
  fi

  if ! cluster_scalable; then
    logger.info 'Skipping scaling up, the cluster is not scalable.'
    return 0
  fi

  logger.info "Scaling cluster to ${SCALE_UP}"

  logger.debug 'Get the machineset with most replicas'
  current_total="$(oc get machineconfigpool worker -o jsonpath='{.status.readyMachineCount}')"
  az_total="$(oc get machineset -n openshift-machine-api --no-headers|wc -l)"

  logger.debug "ready machine count: ${current_total}, number of available zones: ${az_total}"

  if [[ "${SCALE_UP}" == "${current_total}" ]]; then
    logger.success "Cluster is already scaled up to ${SCALE_UP} replicas"
    return 0
  fi

  idx=0
  for mset in $(oc get machineset -n openshift-machine-api -o name); do
    replicas=$(( SCALE_UP / az_total ))
    if [ ${idx} -lt $(( SCALE_UP % az_total )) ];then
      (( replicas++ )) || true
    fi
    (( idx++ )) || true
    logger.debug "Bump ${mset} to ${replicas}"
    oc scale "${mset}" -n openshift-machine-api --replicas="${replicas}"
  done
  wait_until_machineset_scales_up "${SCALE_UP}"
}

# Waits until worker nodes scale up to the desired number of replicas
# Parameters: $1 - desired number of replicas
function wait_until_machineset_scales_up {
  logger.info "Waiting until worker nodes scale up to $1 replicas"
  local available
  for _ in {1..150}; do  # timeout after 15 minutes
    available=$(oc get machineconfigpool worker -o jsonpath='{.status.readyMachineCount}')
    if [[ ${available} -eq $1 ]]; then
      echo ''
      logger.success "successfully scaled up to $1 replicas"
      return 0
    fi
    echo -n "."
    sleep 6
  done
  echo -e "\n\n"
  logger.error "Timeout waiting for scale up to $1 replicas"
  return 1
}

function cluster_scalable {
  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') = VSphere ]]; then
    return 1
  fi
}
