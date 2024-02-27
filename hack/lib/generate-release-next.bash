#!/usr/bin/env bash

function generate_release_next(){
  header "Generate release next artifacts"

  local serving_dir eventing_dir ekb_dir
  serving_dir=$(mktemp -d)
  eventing_dir=$(mktemp -d)
  ekb_dir=$(mktemp -d)

  git clone --branch release-next https://github.com/openshift-knative/serving.git "$serving_dir"
  KNATIVE_SERVING_MANIFESTS_DIR="${serving_dir}/openshift/release/artifacts"
  export KNATIVE_SERVING_MANIFESTS_DIR

  git clone --branch release-next https://github.com/openshift-knative/eventing.git "$eventing_dir"
  KNATIVE_EVENTING_MANIFESTS_DIR="${eventing_dir}/openshift/release/artifacts"
  export KNATIVE_EVENTING_MANIFESTS_DIR

  git clone --branch release-next https://github.com/openshift-knative/eventing-kafka-broker.git "$ekb_dir"
  KNATIVE_EVENTING_KAFKA_BROKER_MANIFESTS_DIR="${ekb_dir}/openshift/release/artifacts"
  export KNATIVE_EVENTING_KAFKA_BROKER_MANIFESTS_DIR

  export GOPATH=/tmp/go

  #pushd $operator_dir || return $?
  #export ON_CLUSTER_BUILDS=true
  #export DOCKER_REPO_OVERRIDE=image-registry.openshift-image-registry.svc:5000/openshift-marketplace
  make generated-files #images

#  OPENSHIFT_CI="true" TRACING_BACKEND="zipkin" ENABLE_TRACING="true" make generated-files images install-tracing install-eventing || failed=$?
  cat olm-catalog/serverless-operator/manifests/serverless-operator.clusterserviceversion.yaml
}
