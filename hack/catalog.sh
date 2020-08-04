#!/usr/bin/env bash

NAME="serverless-operator"
DISPLAYNAME="Serverless Operator"

# Determine if we're running locally or in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  # HACK: Until this is built properly.
  export IMAGE_SERVERLESS_INDEX="docker.io/markusthoemmes/openshift-serverless-bundle:latest"
elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
  export IMAGE_SERVERLESS_INDEX="${DOCKER_REPO_OVERRIDE}/openshift-serverless-bundle:latest"
else
  export IMAGE_SERVERLESS_INDEX="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:serverless-bundle"
fi

cat <<EOF | sed 's/^  *$//'
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
        image: quay.io/joelanford/example-operator-index:0.1.0
        command:
        - /bin/sh
        - -c
        - |-
          mkdir -p /database && \
          /bin/opm registry add   -d /database/index.db --mode=replaces -b docker.io/markusthoemmes/serverless-index:1.7.2 && \
          /bin/opm registry add   -d /database/index.db --mode=replaces -b $IMAGE_SERVERLESS_INDEX && \
          /bin/opm registry serve -d /database/index.db -p 50051
---
apiVersion: v1
kind: Service
metadata:
  name: serverless-index
  labels:
    app: serverless-index
spec:
  ports:
  - name: grpc
    port: 50051
  selector:
    app: serverless-index
  type: ClusterIP
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: $NAME
  annotations:
    operators.operatorframework.io/injected-bundles: '["$IMAGE_SERVERLESS_INDEX"]'
spec:
  address: serverless-index.openshift-marketplace:50051
  displayName: $DISPLAYNAME
  publisher: Red Hat
  sourceType: grpc
EOF
