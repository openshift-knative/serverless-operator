#!/usr/bin/env bash

function ensure_catalogsource_installed {
  install_catalogsource
}

function install_catalogsource {
  logger.info "Installing CatalogSource"
  local rootdir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"

  # HACK: Allow to run the image as root
  oc adm policy add-scc-to-user anyuid -z default -n "$OLM_NAMESPACE"

  csv="${rootdir}/olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml"
  mv "$csv" "${rootdir}/raw.yaml"

  # Determine if we're running locally or in CI.
  if [ -n "$OPENSHIFT_CI" ]; then
    export IMAGE_KNATIVE_OPERATOR="${IMAGE_FORMAT//\$\{component\}/knative-operator}"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${IMAGE_FORMAT//\$\{component\}/knative-openshift-ingress}"
  elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
    export IMAGE_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/knative-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress"
  else
    export IMAGE_KNATIVE_OPERATOR="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:knative-operator"
    export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:knative-openshift-ingress"
  fi

  rm "$csv"
  cat "${rootdir}/raw.yaml" | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' > "$csv"

  cat "$csv"

  oc -n "$OLM_NAMESPACE" new-build --binary --strategy=docker --name serverless-bundle
  oc -n "$OLM_NAMESPACE" start-build serverless-bundle --from-dir olm-catalog/serverless-operator -F

  mv "${rootdir}/raw.yaml" "$csv"

  # Install the index deployment
  cat <<EOF | oc apply -f - || return $?
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
          podman login -u kubeadmin -p "$(oc whoami -t)" --tls-verify=false image-registry.openshift-image-registry.svc:5000
          mkdir -p /database && \
          /bin/opm registry add                         -d /database/index.db --mode=replaces -b docker.io/markusthoemmes/serverless-bundle:1.7.2
          /bin/opm registry add --container-tool=podman -d /database/index.db --mode=replaces -b image-registry.openshift-image-registry.svc:5000/$OLM_NAMESPACE/serverless-bundle && \
          /bin/opm registry serve -d /database/index.db -p 50051
EOF

  wait_until_pods_running "$OLM_NAMESPACE"
  indexip="$(oc -n "$OLM_NAMESPACE" get pods -l app=serverless-index -ojsonpath='{.items[0].status.podIP}')"

  # Install the catalogsource
  cat <<EOF | oc apply -f - || return $? 
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
