#!/usr/bin/env bash

DIR=$(cd $(dirname "$0")/.. && pwd)
CRD_DIR=$DIR/.crds              # scratch dir

OLM_DIR=${OLM_DIR:-$DIR/olm-catalog}
NAME=${NAME:-$(ls $OLM_DIR)}

x=( $(echo $NAME | tr '-' ' ') )
DISPLAYNAME=${DISPLAYNAME:=${x[*]^}}

indent() {
  INDENT="      "
  ENDASH="    - "
  sed "s/^/$INDENT/" | sed "s/^${INDENT}\($1\)/${ENDASH}\1/"
}

# initialize scratch dir
rm -rf $CRD_DIR
mkdir $CRD_DIR

# deal with identical CRD's in nested dirs: highest version wins
find $OLM_DIR -name '*_crd.yaml' | sort -n | xargs -I{} cp {} $CRD_DIR/

# Determine if we're running locally or in CI.
if [ -n "$OPENSHIFT_CI" ]; then
  export IMAGE_KNATIVE_OPERATOR="${IMAGE_FORMAT//\$\{component\}/knative-operator}"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${IMAGE_FORMAT//\$\{component\}/knative-openshift-ingress}"
elif [ -n "$DOCKER_REPO_OVERRIDE" ]; then
  export IMAGE_KNATIVE_OPERATOR="${DOCKER_REPO_OVERRIDE}/knative-operator"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="${DOCKER_REPO_OVERRIDE}/knative-openshift-ingress"
else
  export IMAGE_KNATIVE_OPERATOR="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0-rc1:knative-operator"
  export IMAGE_KNATIVE_OPENSHIFT_INGRESS="registry.svc.ci.openshift.org/openshift/openshift-serverless-v1.8.0-rc1:knative-openshift-ingress"
fi

CRD=$(cat $(ls $CRD_DIR/*) | grep -v -- "---" | indent apiVersion)
CSV=$(cat $(find $OLM_DIR -name '*version.yaml' | sort -n) | envsubst '$IMAGE_KNATIVE_OPERATOR $IMAGE_KNATIVE_OPENSHIFT_INGRESS' | indent apiVersion)
PKG=$(cat $OLM_DIR/$NAME/*package.yaml | indent packageName)

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
      labels: app: serverless-index
    spec:
      containers:
      - name: registry
        image: quay.io/joelanford/example-operator-index:0.1.0
        command:
        - /bin/sh
        - -c
        - |-
          mkdir -p /database && \
          /bin/opm registry add   -d /database/index.db --mode=replaces -b quay.io/bbrowning/openshift-serverless-bundle:latest && \
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
    operators.operatorframework.io/injected-bundles: '["quay.io/bbrowning/openshift-serverless-bundle:latest"]'
spec:
  address: serverless-bundle
  displayName: $DISPLAYNAME
  publisher: Red Hat
  sourceType: grpc
EOF
