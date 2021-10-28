#!/usr/bin/env bash

set -Eeuo pipefail

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/lib/__sources__.bash"

NAME="${NAME:-hello}"
TARGET="${USER:-world}"

# Create a sample Knative Service
cat <<EOF | oc apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: ${NAME}
spec:
  template:
    spec:
      containers:
        - image: gcr.io/knative-samples/helloworld-go
          env:
            - name: TARGET
              value: ${TARGET}
          readinessProbe:
            httpGet:
              path: /
EOF

# Wait for the Knative Service to be ready
timeout 100 "[[ \$(oc get ksvc ${NAME} -o \
jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') != 'True' ]]"

# Get the URL from the knative service
URL="$(oc get ksvc hello -o jsonpath='{.status.url}')"

# Fetch it, accounting for possible ingress race conditions
until curl -f "$URL" 2>/dev/null; do sleep 2; done
