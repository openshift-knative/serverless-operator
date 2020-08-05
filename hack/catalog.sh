#!/usr/bin/env bash

NAME="serverless-operator"
DISPLAYNAME="Serverless Operator"

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
        image: docker.io/markusthoemmes/serverless-index:registry10
        command:
        - /bin/sh
        - -c
        - |-
          podman login -u kubeadmin -p "$(oc whoami -t)" --tls-verify=false image-registry.openshift-image-registry.svc:5000
          mkdir -p /database && \
          /bin/opm registry add --container-tool=podman -d /database/index.db --mode=replaces -b image-registry.openshift-image-registry.svc:5000/openshift-marketplace/serverless-bundle && \
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
spec:
  address: serverless-index.openshift-marketplace:50051
  displayName: $DISPLAYNAME
  publisher: Red Hat
  sourceType: grpc
EOF
