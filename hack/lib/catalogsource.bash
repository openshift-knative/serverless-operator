#!/usr/bin/env bash

function ensure_catalogsource_installed {
  logger.info 'Check if CatalogSource is installed'
  if oc get catalogsource "$OPERATOR" -n "$OLM_NAMESPACE" > /dev/null 2>&1; then
    logger.success 'CatalogSource is already installed.'
    return 0
  fi
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"

  local rootdir pull_user

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Add a user that is allowed to pull images from the registry.
  pull_user="puller"
  add_user "$pull_user" "puller"
  oc -n "$OLM_NAMESPACE" policy add-role-to-user registry-viewer "$pull_user"
  token=$(oc --kubeconfig=${pull_user}.kubeconfig whoami -t)

  csv="${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"

  logger.debug "Create a backup of the CSV so we don't pollute the repository."
  mkdir -p "${rootdir}/_output"
  cp "$csv" "${rootdir}/_output/bkp.yaml"

  if [ -n "$OPENSHIFT_CI" ]; then
    # Image variables supplied by ci-operator.
    sed -i "s,image: .*openshift-serverless-.*:knative-operator,image: ${KNATIVE_OPERATOR}," "$csv"
    sed -i "s,image: .*openshift-serverless-.*:knative-openshift-ingress,image: ${KNATIVE_OPENSHIFT_INGRESS}," "$csv"
    sed -i "s,image: .*openshift-serverless-.*:openshift-knative-operator,image: ${OPENSHIFT_KNATIVE_OPERATOR}," "$csv"
  elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    sed -i "s,image: .*openshift-serverless-.*:knative-operator,image: ${DOCKER_REPO_OVERRIDE}/knative-operator," "$csv"
    sed -i "s,image: .*openshift-serverless-.*:knative-openshift-ingress,image: ${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress," "$csv"
    sed -i "s,image: .*openshift-serverless-.*:openshift-knative-operator,image: ${DOCKER_REPO_OVERRIDE}/openshift-knative-operator," "$csv"
  fi

  if [ -n "$OPENSHIFT_CI" ] || [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    logger.info 'Listing CSV content'
    cat "$csv"
  fi

  if ! oc get buildconfigs serverless-bundle -n "$OLM_NAMESPACE" >/dev/null 2>&1; then
    logger.info 'Create a bundle image build'
    oc -n "${OLM_NAMESPACE}" new-build --binary \
      --strategy=docker --name serverless-bundle
  else
    logger.info 'Serverless bundle image build is already created'
  fi

  # Fetch previously created ConfigMap or remove empty file
  oc -n "${OLM_NAMESPACE}" get configmap serverless-bundle-sha1sums \
    -o jsonpath='{.data.serverless-bundle\.sha1sum}' \
    > "${rootdir}/_output/serverless-bundle.sha1sum" 2>/dev/null \
    || rm -f "${rootdir}/_output/serverless-bundle.sha1sum"

  if ! [ -f "${rootdir}/_output/serverless-bundle.sha1sum" ] || \
      ! sha1sum --check --status "${rootdir}/_output/serverless-bundle.sha1sum"; then
    logger.info 'Build the bundle image in the cluster-internal registry.'
    oc -n "${OLM_NAMESPACE}" start-build serverless-bundle \
      --from-dir "${rootdir}/olm-catalog/serverless-operator" -F
    mkdir -p "${rootdir}/_output"
    find "${rootdir}/olm-catalog/serverless-operator" -type f -exec sha1sum {} + \
      > "${rootdir}/_output/serverless-bundle.sha1sum"
    oc -n "${OLM_NAMESPACE}" delete configmap serverless-bundle-sha1sums --ignore-not-found=true
    oc -n "${OLM_NAMESPACE}" create configmap serverless-bundle-sha1sums \
      --from-file="${rootdir}/_output/serverless-bundle.sha1sum"
    rm -f "${rootdir}/_output/serverless-bundle.sha1sum"
  else
    logger.info 'Serverless bundle build is up-to-date.'
  fi

  logger.debug 'Undo potential changes to the CSV to not pollute the repository.'
  mv "${rootdir}/_output/bkp.yaml" "$csv"

  logger.debug "HACK: Allow to run the index pod as privileged so it has \
necessary access to run the podman commands."
  oc -n "$OLM_NAMESPACE" adm policy add-scc-to-user privileged -z default

  logger.info 'Install the index deployment.'
  # This image was built using the Dockerfile at 'olm-catalog/serverless-operator/index.Dockerfile'.
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f -
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serverless-index
  labels:
    app: serverless-index
spec:
  selector:
    matchLabels:
      app: serverless-index
  template:
    metadata:
      labels:
        app: serverless-index
    spec:
      containers:
      - name: registry
        image: quay.io/openshift-knative/serverless-index:v1.14.3
        securityContext:
          privileged: true
        ports:
        - containerPort: 50051
          name: grpc
          protocol: TCP
        readinessProbe:
          exec:
            command:
            - grpc_health_probe
            - -addr=localhost:50051
        command:
        - /bin/sh
        - -c
        - |-
          podman login -u $pull_user -p $token image-registry.openshift-image-registry.svc:5000 && \
          /bin/opm registry add -d index.db --container-tool=podman --mode=replaces -b quay.io/openshift-knative/serverless-bundle:1.7.2,quay.io/openshift-knative/serverless-bundle:1.8.0,quay.io/openshift-knative/serverless-bundle:1.9.0,quay.io/openshift-knative/serverless-bundle:1.10.0,quay.io/openshift-knative/serverless-bundle:1.10.1,quay.io/openshift-knative/serverless-bundle:1.11.0,registry.ci.openshift.org/openshift/openshift-serverless-v1.12.0:serverless-bundle,registry.ci.openshift.org/openshift/openshift-serverless-v1.13.0:serverless-bundle,registry.ci.openshift.org/openshift/openshift-serverless-v1.14.0:serverless-bundle,image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-bundle && \
          /bin/opm registry serve -d index.db -p 50051
---
apiVersion: v1
kind: Service
metadata:
  name: serverless-index
  labels:
    app: serverless-index
spec:
  selector:
    app: serverless-index
  ports:
    - protocol: TCP
      port: 50051
      targetPort: 50051
EOF

  logger.info 'Wait for the index pod to be up to avoid inconsistencies with the catalog source.'
  wait_until_labelled_pods_are_ready app=serverless-index "$OLM_NAMESPACE"

  logger.info 'Install the catalogsource.'
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: serverless-operator
spec:
  address: serverless-index.${OLM_NAMESPACE}.svc:50051
  displayName: "Serverless Operator"
  publisher: Red Hat
  sourceType: grpc
EOF

  logger.success "CatalogSource installed successfully"
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource $OPERATOR"
  oc delete catalogsource --ignore-not-found=true -n "$OLM_NAMESPACE" "$OPERATOR"
  oc delete service --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete deployment --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-index
  oc delete configmap --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-bundle-sha1sums
  oc delete buildconfig --ignore-not-found=true -n "$OLM_NAMESPACE" serverless-bundle
  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 300 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]"
  logger.success 'CatalogSource deleted'
}

# TODO: Deduplicate with the `create_htpasswd_users` function in test/lib.bash.
function add_user {
  local name pass
  name=${1:?Pass a username as arg[1]}
  pass=${2:?Pass a password as arg[2]}

  if oc get user "$name" >/dev/null 2>&1; then
    logger.success "User ${name} already exists."
    return 0
  fi

  logger.info "Creating user $name:***"
  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.debug 'Secret htpass-secret already existed, updating it.'
    # Add a newline to the end of the file if not already present (htpasswd will butcher it otherwise).
    [ -n "$(tail -c1 users.htpasswd)" ] && echo >> users.htpasswd
  else
    touch users.htpasswd
  fi

  htpasswd -b users.htpasswd "$name" "$pass"

  kubectl create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | kubectl apply -f -

  if oc get oauth.config.openshift.io cluster > /dev/null 2>&1; then
    oc replace -f openshift/identity/htpasswd.yaml
  else
    oc apply -f openshift/identity/htpasswd.yaml
  fi

  logger.debug 'Generate kubeconfig'
  cp "${KUBECONFIG}" "$name.kubeconfig"
  occmd="bash -c '! oc login --kubeconfig=${name}.kubeconfig --username=${name} --password=${pass} > /dev/null'"
  timeout 180 "${occmd}"
}
