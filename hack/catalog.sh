#!/usr/bin/env bash

NAME="serverless-operator"
DISPLAYNAME="Serverless Operator"

# Determine if we're running locally or in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  # HACK: Until this is built properly.
  export IMAGE_SERVERLESS_INDEX="${IMAGE_FORMAT//\$\{component\}/serverless-bundle}"
elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
  export IMAGE_SERVERLESS_INDEX="${DOCKER_REPO_OVERRIDE}/openshift-serverless-bundle:latest"
else
  export IMAGE_SERVERLESS_INDEX="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0:serverless-bundle"
fi

cat <<EOF | sed 's/^  *$//'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serverless-registry
spec:
  selector:
    matchLabels:
      app: serverless-registry
  template:
    metadata:
      labels:
        app: serverless-registry
    spec:
      containers:
      - name: registry
        image: $IMAGE_SERVERLESS_INDEX
        command:
        - /usr/bin/registry-server
        - --database=/bundle/bundles.db
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
---
apiVersion: v1
kind: Service
metadata:
  name: serverless-registry
  labels:
    app: serverless-registry
spec:
  ports:
  - name: grpc
    port: 50051
  selector:
    app: serverless-registry
  type: ClusterIP
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: $NAME
spec:
  address: serverless-registry.openshift-marketplace:50051
  displayName: $DISPLAYNAME
  publisher: Red Hat
  sourceType: grpc
EOF
