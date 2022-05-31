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
    override_storage_version_migration_images "$csv"
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
          /bin/opm registry add -d index.db --container-tool=podman --mode=replaces -b registry.ci.openshift.org/openshift/openshift-serverless-v$PREVIOUS_VERSION:serverless-stop-bundle,image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-bundle && \
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
  oc wait deployment -n "$OLM_NAMESPACE" serverless-index --for=condition=Available --timeout=600s

  logger.info 'Install the catalogsource.'
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${OPERATOR}
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

function add_user {
  local name pass occmd rootdir
  name=${1:?Pass a username as arg[1]}
  pass=${2:?Pass a password as arg[2]}

  rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  logger.info "Creating user $name:***"
  if oc get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.debug 'Secret htpass-secret already existed, updating it.'
    # Add a newline to the end of the file if not already present (htpasswd will butcher it otherwise).
    [ -n "$(tail -c1 users.htpasswd)" ] && echo >> users.htpasswd
  else
    touch users.htpasswd
  fi

  htpasswd -b users.htpasswd "$name" "$pass"

  oc create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run=client -o yaml | oc apply -f -

  if oc get oauth.config.openshift.io cluster &>/dev/null; then
    oc replace -f "${rootdir}/openshift/identity/htpasswd.yaml"
  else
    oc apply -f "${rootdir}/openshift/identity/htpasswd.yaml"
  fi

  logger.debug 'Generate kubeconfig'

  if oc config current-context >&/dev/null; then
    ctx=$(oc config current-context)
    cluster=$(oc config view -ojsonpath="{.contexts[?(@.name == \"$ctx\")].context.cluster}")
    server=$(oc config view -ojsonpath="{.clusters[?(@.name == \"$cluster\")].cluster.server}")
    logger.debug "Context: $ctx, Cluster: $cluster, Server: $server"
  else
    # Fallback to in-cluster api server service.
    server="https://kubernetes.default.svc"
  fi

  occmd="bash -c '! oc login --kubeconfig=${name}.kubeconfig --insecure-skip-tls-verify=true --username=${name} --password=${pass} ${server} > /dev/null'"
  timeout 600 "${occmd}"

  logger.info "Kubeconfig for user ${name} created"
}

# Use images from internal registry in order to test issues such as SRVCOM-1873
# during upgrades. The issue is only reproducible if Knative version is identical
# before/after upgrade but the migrator Job spec/image changes. Before upgrade,
# the images are pulled from CI registry and after upgrade they're pulled from
# internal registry. This way the Job spec is changed even though Knative version
# remains same.
function override_storage_version_migration_images {
  local csv images name
  csv=${1:?Pass csv as arg[1]}
  # Get all storage version-related images.
  while IFS=$'\n' read -r line; do
    images+=("$line");
  done < <(grep storage-version-migration "$csv" | grep "image:" | awk '{ print $2 }' | awk -F"\"" '{ print $2 }')
  for image_pullspec in "${images[@]}"; do
    name=$(echo "$image_pullspec" | awk -F":" '{ print $2 }')
    oc tag -n "$OPERATORS_NAMESPACE" "$image_pullspec" "${name}:latest" --reference-policy=local
    sed -i "s,${image_pullspec},image-registry.openshift-image-registry.svc:5000/${OPERATORS_NAMESPACE}/${name}," "$csv"
  done
  oc policy add-role-to-group \
    system:image-puller system:serviceaccounts:knative-serving \
    --namespace="$OPERATORS_NAMESPACE"
  oc policy add-role-to-group \
    system:image-puller system:serviceaccounts:knative-eventing \
    --namespace="$OPERATORS_NAMESPACE"
}
