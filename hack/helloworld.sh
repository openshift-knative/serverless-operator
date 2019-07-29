#!/usr/bin/env bash

NAME=hello
TARGET=${USER:-world}

# Create a sample Knative Service
cat <<EOF | kubectl apply -f -
apiVersion: serving.knative.dev/v1alpha1
kind: Service
metadata:
  name: $NAME
spec:
  template:
    spec:
      containers:
        - image: gcr.io/knative-samples/helloworld-go
          env:
            - name: TARGET
              value: $TARGET
EOF

# Wait for the Knative Service to be ready
while output=$(kubectl get ksvc $NAME); do
  echo "$output"
  echo $output | grep True >/dev/null && break
  sleep 2
done

# Parse the URL from the knative service
URL=$(kubectl get ksvc $NAME | grep True | awk '{print $2}')

# Fetch it, accounting for possible istio race conditions
until curl -f $URL; do sleep 2; done
