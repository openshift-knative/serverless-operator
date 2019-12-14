#!/usr/bin/env bash

# This script can be used to publish all images built by
# this repository to the specified docker repository.

repo=$1

docker build -t "$repo/knative-serving-operator" serving/operator
docker push "$repo/knative-serving-operator"

docker build -t "$repo/knative-openshift-ingress" serving/ingress
docker push "$repo/knative-openshift-ingress"

#docker build -t "$repo/knative-networking-openshift" serving/networking-openshift
#docker push "$repo/knative-networking-openshift"
