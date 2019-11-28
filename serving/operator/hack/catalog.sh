#!/usr/bin/env bash

DIR=${DIR:-$(cd $(dirname "$0")/.. && pwd)}
NAME=${NAME:-$(ls $DIR/deploy/olm-catalog)}

x=( $(echo $NAME | tr '-' ' ') )
DISPLAYNAME=${DISPLAYNAME:=${x[*]^}}

indent() {
  INDENT="      "
  ENDASH="    - "
  sed "s/^/$INDENT/" | sed "s/^${INDENT}\($1\)/${ENDASH}\1/"
}

rm -rf $DIR/.crds
mkdir $DIR/.crds
find $DIR/deploy/olm-catalog -name '*_crd.yaml' | sort -n | xargs -I{} cp {} $DIR/.crds/

CRD=$(cat $(ls $DIR/.crds/*) | grep -v -- "---" | indent apiVersion)
CSV=$(cat $(find $DIR/deploy/olm-catalog -name '*version.yaml' | sort -n) | indent apiVersion)
PKG=$(cat $DIR/deploy/olm-catalog/$NAME/*package.yaml | indent packageName)

cat <<EOF | sed 's/^  *$//'
kind: ConfigMap
apiVersion: v1
metadata:
  name: $NAME

data:
  customResourceDefinitions: |-
$CRD
  clusterServiceVersions: |-
$CSV
  packages: |-
$PKG
---
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: $NAME
spec:
  configMap: $NAME
  displayName: $DISPLAYNAME
  publisher: Red Hat
  sourceType: internal
EOF
