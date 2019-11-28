#!/usr/bin/env bash

set -ex

version=$1

for image in {knative-serving-queue,knative-serving-activator,knative-serving-autoscaler,knative-serving-controller,knative-serving-webhook,knative-serving-certmanager,knative-serving-istio}; do
  src=registry.svc.ci.openshift.org/openshift/knative-$version:$image
  tgt=quay.io/openshift-knative/$image:$version
  docker pull $src
  docker tag $src $tgt
  docker push $tgt
done
