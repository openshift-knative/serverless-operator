#!/usr/bin/env bash

function install_catalogsource {
  logger.info "Installing CatalogSource"
  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # Add a user that is allowed to pull images from the registry.
  pull_user="puller"
  add_user "$pull_user" "puller"
  oc -n "$OLM_NAMESPACE" policy add-role-to-user registry-viewer "$pull_user"
  token=$(oc --config=$pull_user.kubeconfig whoami -t)

  csv="${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"
  # Create a backup of the CSV so we don't pollute the repository.
  cp "$csv" "${rootdir}/bkp.yaml"

  if [ -n "$OPENSHIFT_CI" ]; then
    sed -i "s,image: .*:knative-operator,image: ${IMAGE_FORMAT//\$\{component\}/knative-operator}," "$csv"
    sed -i "s,image: .*:knative-openshift-ingress,image: ${IMAGE_FORMAT//\$\{component\}/knative-openshift-ingress}," "$csv"
  elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    sed -i "s,image: .*:knative-operator,image: ${DOCKER_REPO_OVERRIDE}/knative-operator," "$csv"
    sed -i "s,image: .*:knative-openshift-ingress,image: ${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress," "$csv"
  fi

  cat "$csv"

  # Build the bundle image in the cluster-internal registry.
  oc -n "$OLM_NAMESPACE" new-build --binary --strategy=docker --name serverless-bundle
  oc -n "$OLM_NAMESPACE" start-build serverless-bundle --from-dir olm-catalog/serverless-operator -F

  # Undo potential changes to the CSV to not pollute the repository.
  mv "${rootdir}/bkp.yaml" "$csv"

  # HACK: Allow to run the index pod as root so it can create the necessary directories.
  oc -n "$OLM_NAMESPACE" adm policy add-scc-to-user anyuid -z default

  # Install the index deployment.
  # TODO: Fix the --skip-tls bugs in operator-registry to not have to rely on a self-built
  # image here. This has been built from
  # https://github.com/markusthoemmes/operator-registry/tree/hack-ignore-tls.
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f - || return $?
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serverless-index
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
        image: docker.io/markusthoemmes/serverless-index:registry10
        ports:
        - containerPort: 50051
          name: grpc
          protocol: TCP
        livenessProbe:
          exec:
            command:
            - grpc_health_probe
            - -addr=localhost:50051
        readinessProbe:
          exec:
            command:
            - grpc_health_probe
            - -addr=localhost:50051
        command:
        - /bin/sh
        - -c
        - |-
          podman login -u $pull_user -p $token --tls-verify=false image-registry.openshift-image-registry.svc:5000
          mkdir -p /database && \
          /bin/opm registry add                         -d /database/index.db --mode=replaces -b docker.io/markusthoemmes/serverless-bundle:1.7.2
          /bin/opm registry add --container-tool=podman -d /database/index.db --mode=replaces -b image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-bundle && \
          /bin/opm registry serve -d /database/index.db -p 50051
EOF

  # Wait for the index pod to be up to avoid inconsistencies with the catalog source.
  wait_until_pods_running "$OLM_NAMESPACE"
  indexip="$(oc -n "$OLM_NAMESPACE" get pods -l app=serverless-index -ojsonpath='{.items[0].status.podIP}')"

  # Install the catalogsource.
  cat <<EOF | oc apply -n "$OLM_NAMESPACE" -f - || return $? 
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: serverless-operator
spec:
  address: $indexip:50051
  displayName: "Serverless Operator"
  publisher: Red Hat
  sourceType: grpc
EOF

  logger.success "CatalogSource installed successfully"
}

function delete_catalog_source {
  logger.info "Deleting CatalogSource $OPERATOR"
  oc delete catalogsource --ignore-not-found=true -n "$OLM_NAMESPACE" "$OPERATOR" || return 10
  [ -f "$CATALOG_SOURCE_FILENAME" ] && rm -v "$CATALOG_SOURCE_FILENAME"
  logger.info "Wait for the ${OPERATOR} pod to disappear"
  timeout 900 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -gt 0 ]]" || return 11
  logger.success 'CatalogSource deleted'
}

# TODO: Deduplicate with the `create_htpasswd_users` function in test/lib.bash.
function add_user {
  name=$1
  pass=$2

  logger.info "Creating user $name:$pass"
  if kubectl get secret htpass-secret -n openshift-config -o jsonpath='{.data.htpasswd}' 2>/dev/null | base64 -d > users.htpasswd; then
    logger.info 'Secret htpass-secret already existsed, updating it.'
    sed -i -e '$a\' users.htpasswd
  else
    touch users.htpasswd
  fi

  htpasswd -b users.htpasswd "$name" "$pass"

  kubectl create secret generic htpass-secret \
    --from-file=htpasswd="$(pwd)/users.htpasswd" \
    -n openshift-config \
    --dry-run -o yaml | kubectl apply -f -
  oc apply -f openshift/identity/htpasswd.yaml

  logger.info 'Generate kubeconfig'
  cp "${KUBECONFIG}" "$name.kubeconfig"
  occmd="bash -c '! oc login --config=$name.kubeconfig --username=$name --password=$pass > /dev/null'"
  timeout 900 "${occmd}" || return 1
}