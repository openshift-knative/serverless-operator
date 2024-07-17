#!/usr/bin/env bash

function scale_up_workers {
  local current_total az_total replicas idx ignored_zone
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

  # WORKAROUND: OpenShift in CI cannot scale machine set in us-east-1e zone and throws:
  # Your requested instance type (m5.xlarge) is not supported in your requested Availability Zone
  # (us-east-1e). Please retry your request by not specifying an Availability Zone
  # or choosing us-east-1a, us-east-1b, us-east-1c, us-east-1d, us-east-1f."
  ignored_zone="us-east-1e"
  az_total="$(oc get machineset -n openshift-machine-api -oname | grep -cv "$ignored_zone")"

  logger.debug "ready machine count: ${current_total}, number of available zones: ${az_total}"

  if [[ "${SCALE_UP}" == "${current_total}" ]]; then
    logger.success "Cluster is already scaled up to ${SCALE_UP} replicas"
    return 0
  fi

  idx=0
  for mset in $(oc get machineset -n openshift-machine-api -o name | grep -v "$ignored_zone"); do
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
  if ! oc get machineconfigpool &>/dev/null; then
    return 1
  fi
  # Prevent scaling for single-node OpenShift.
  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.controlPlaneTopology}') == SingleReplica && \
        $(oc get infrastructure cluster -ojsonpath='{.status.infrastructureTopology}') == SingleReplica ]]; then
    return 1
  fi
  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') = VSphere ]]; then
    return 1
  fi
  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platformStatus.aws.resourceTags[?(@.key=="red-hat-clustertype")].value}') = rosa ]]; then
    return 1
  fi
  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platformStatus.aws.resourceTags[?(@.key=="red-hat-clustertype")].value}') = osd ]]; then
    return 1
  fi
}

# Convert existing machinesets to spot instances
function use_spot_instances {
  if ! cluster_scalable; then
    logger.info 'Skipping spot instances, the cluster is not scalable.'
    return
  fi

  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platform}') != AWS ]]; then
    logger.info "Skipping spot instances. Spot instances only supported on AWS."
    return
  fi

  if [ -z "$OPENSHIFT_CI" ] ; then
    logger.info "Skipping spot instances for non-CI runs."
    return
  fi

#  if ! echo $JOB_SPEC | grep -q "type:periodic"; then
#    logger.info "Skipping spot instances. Not periodic runs."
#    return
#  fi

  logger.info "Convert MachineSets to spot instances"

  for mset in $(oc get machineset -oname); do
  	mset_name=$(oc get "${mset}" -ojsonpath='{.metadata.name}')

  	# New machine set has a specific suffix
  	mset_name="${mset_name}-s"

  	# Create the new machineset with spotMarketOptions
  	oc get "${mset}" -ojson | \
  		jq '.metadata.name |= "'${mset_name}'"' | \
  		jq '.spec.selector.matchLabels."machine.openshift.io/cluster-api-machineset" |= "'${mset_name}'"' | \
  		jq '.spec.template.metadata.labels."machine.openshift.io/cluster-api-machineset" |= "'${mset_name}'"' | \
  		jq '.spec.template.spec.providerSpec.value.spotMarketOptions |= {}' | oc create -f -

  	# Delete the old machineset
  	oc delete "${mset}"
  done
}
