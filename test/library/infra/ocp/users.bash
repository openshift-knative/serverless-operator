#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash
include infra/await.bash

function create_htpasswd_users {
  local occmd num_users
  num_users=2
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
