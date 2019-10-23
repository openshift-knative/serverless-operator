#!/usr/bin/env bash

function install_catalogsource {
  logger.info "Installing CatalogSource"

  local operator_image
  operator_image=$(tag_operator_image)
  if [[ -n "${operator_image}" ]]; then
    ./hack/catalog.sh | sed -e "s+\(.* containerImage:\)\(.*\)+\1 ${operator_image}+g" > "$CATALOG_SOURCE_FILENAME"
  else
    ./hack/catalog.sh > "$CATALOG_SOURCE_FILENAME"
  fi
  oc apply -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME" || return 1

  logger.success "CatalogSource installed successfully"
}

function tag_operator_image {
  if [[ -n "${OPENSHIFT_BUILD_NAMESPACE:-}" ]]; then
    oc policy add-role-to-group system:image-puller "system:serviceaccounts:${OPERATORS_NAMESPACE}" --namespace="${OPENSHIFT_BUILD_NAMESPACE}" >/dev/null
    oc tag --insecure=false -n "${OPERATORS_NAMESPACE}" "${OPENSHIFT_REGISTRY}/${OPENSHIFT_BUILD_NAMESPACE}/stable:${OPERATOR} ${OPERATOR}:latest" >/dev/null
    echo "$INTERNAL_REGISTRY/$OPERATORS_NAMESPACE/$OPERATOR"
  fi
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource"
  oc delete --ignore-not-found=true -n "$OPERATORS_NAMESPACE" -f "$CATALOG_SOURCE_FILENAME"
  rm -v "$CATALOG_SOURCE_FILENAME"
}

function dump_openshift_olm_state {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces
  logger.info "Dump of catalog operator log"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator
}

function dump_openshift_ingress_state {
  logger.info "Dump of routes.route.openshift.io"
  oc get routes.route.openshift.io -o yaml --all-namespaces
  logger.info "Dump of routes.serving.knative.dev"
  oc get routes.serving.knative.dev -o yaml --all-namespaces
  logger.info "Dump of openshift-ingress log"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE"
}

function dump_state {
  if (( INTERACTIVE )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi
  logger.info 'Environment'
  env
  
  dump_cluster_state
  dump_openshift_olm_state
  dump_openshift_ingress_state
}

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
function wait_until_machineset_scales_up {
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

function create_htpasswd_users {
  local occmd num_users
  num_users=3
  logger.info "Creating htpasswd for ${num_users} users"

  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.info 'Secret htpass-secret already existsed, updating it.'
  else
    touch users.htpasswd
  fi

  logger.info 'Add users to htpasswd'
  for i in $(seq 1 $num_users); do
    htpasswd -b users.htpasswd "user${i}" "password${i}"
  done

  kubectl create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run -o yaml | kubectl apply -f -
  oc apply -f openshift/identity/htpasswd.yaml

  logger.info 'Generate kubeconfig for each user'
  for i in $(seq 1 $num_users); do
    cp "${KUBECONFIG}" "user${i}.kubeconfig"
    occmd="bash -c '! oc login --config=user${i}.kubeconfig --username=user${i} --password=password${i} > /dev/null'"
    timeout 900 "${occmd}" || return 1
  done
}

function add_roles {
  logger.info "Adding roles to users"
  oc adm policy add-role-to-user edit user1 -n "$TEST_NAMESPACE"
  oc adm policy add-role-to-user view user2 -n "$TEST_NAMESPACE"
}

function delete_users {
  local user
  logger.info "Deleting users"
  while IFS= read -r line; do
    logger.debug "htpasswd user line: ${line}"
    user=$(echo "${line}" | cut -d: -f1)
    if [ -f "${user}.kubeconfig" ]; then
      rm -v "${user}.kubeconfig"
    fi
  done < "users.htpasswd"
  rm -v users.htpasswd
}
