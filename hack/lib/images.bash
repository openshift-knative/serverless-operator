#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/metadata.bash"

registry_host='registry.ci.openshift.org'
registry="${registry_host}/openshift"

function default_knative_serving_images() {
  local serving
  serving="${registry}/knative-serving"
  local tag
  tag=$(metadata.get dependencies.serving)
  export KNATIVE_SERVING_QUEUE=${KNATIVE_SERVING_QUEUE:-"${serving}-queue:${tag}"}
  export KNATIVE_SERVING_ACTIVATOR=${KNATIVE_SERVING_ACTIVATOR:-"${serving}-activator:${tag}"}
  export KNATIVE_SERVING_AUTOSCALER=${KNATIVE_SERVING_AUTOSCALER:-"${serving}-autoscaler:${tag}"}
  export KNATIVE_SERVING_AUTOSCALER_HPA=${KNATIVE_SERVING_AUTOSCALER_HPA:-"${serving}-autoscaler-hpa:${tag}"}
  export KNATIVE_SERVING_CONTROLLER=${KNATIVE_SERVING_CONTROLLER:-"${serving}-controller:${tag}"}
  export KNATIVE_SERVING_WEBHOOK=${KNATIVE_SERVING_WEBHOOK:-"${serving}-webhook:${tag}"}
  export KNATIVE_SERVING_STORAGE_VERSION_MIGRATION=${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION:-"${serving}-storage-version-migration:${tag}"}
}

function default_knative_eventing_images() {
  local eventing
  eventing="${registry}/knative-eventing"
  local tag
  tag=$(metadata.get dependencies.eventing)
  export KNATIVE_EVENTING_CONTROLLER=${KNATIVE_EVENTING_CONTROLLER:-"${eventing}-controller:${tag}"}
  export KNATIVE_EVENTING_WEBHOOK=${KNATIVE_EVENTING_WEBHOOK:-"${eventing}-webhook:${tag}"}
  export KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION=${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION:-"${eventing}-migrate:${tag}"}
  export KNATIVE_EVENTING_MTBROKER_INGRESS=${KNATIVE_EVENTING_MTBROKER_INGRESS:-"${eventing}-mtbroker-ingress:${tag}"}
  export KNATIVE_EVENTING_MTBROKER_FILTER=${KNATIVE_EVENTING_MTBROKER_FILTER:-"${eventing}-mtbroker-filter:${tag}"}
  export KNATIVE_EVENTING_MTCHANNEL_BROKER=${KNATIVE_EVENTING_MTCHANNEL_BROKER:-"${eventing}-mtchannel-broker:${tag}"}
  export KNATIVE_EVENTING_MTPING=${KNATIVE_EVENTING_MTPING:-"${eventing}-mtping:${tag}"}
  export KNATIVE_EVENTING_CHANNEL_CONTROLLER=${KNATIVE_EVENTING_CHANNEL_CONTROLLER:-"${eventing}-channel-controller:${tag}"}
  export KNATIVE_EVENTING_CHANNEL_DISPATCHER=${KNATIVE_EVENTING_CHANNEL_DISPATCHER:-"${eventing}-channel-dispatcher:${tag}"}
  export KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER=${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER:-"${eventing}-apiserver-receive-adapter:${tag}"}

  export KNATIVE_EVENTING_APPENDER=${KNATIVE_EVENTING_APPENDER:-"${eventing}-appender:${tag}"}
  export KNATIVE_EVENTING_EVENT_DISPLAY=${KNATIVE_EVENTING_EVENT_DISPLAY:-"${eventing}-event-display:${tag}"}
  export KNATIVE_EVENTING_HEARTBEATS_RECEIVER=${KNATIVE_EVENTING_HEARTBEATS_RECEIVER:-"${eventing}-heartbeats-receiver:${tag}"}
  export KNATIVE_EVENTING_MIGRATE=${KNATIVE_EVENTING_MIGRATE:-"${eventing}-migrate:${tag}"}
  export KNATIVE_EVENTING_PONG=${KNATIVE_EVENTING_PONG:-"${eventing}-pong:${tag}"}
  export KNATIVE_EVENTING_SCHEMA=${KNATIVE_EVENTING_SCHEMA:-"${eventing}-schema:${tag}"}
  export KNATIVE_EVENTING_WEBSOCKETSOURCE=${KNATIVE_EVENTING_WEBSOCKETSOURCE:-"${eventing}-websocketsource:${tag}"}
  
  # quay.io multiarch images:
  tag="${tag/knative-/}"
  export KNATIVE_EVENTING_HEARTBEATS=${KNATIVE_EVENTING_HEARTBEATS:-"quay.io/openshift-knative/eventing/heartbeats:${tag}"}
}

function default_knative_eventing_test_images() {
  local eventing
  eventing="quay.io/openshift-knative/eventing"
  local tag
  tag=$(metadata.get dependencies.eventing)
  tag="${tag/knative-/}"

  export KNATIVE_EVENTING_TEST_EVENT_SENDER=${KNATIVE_EVENTING_TEST_EVENT_SENDER:-"${eventing}/event-sender:${tag}"}
  export KNATIVE_EVENTING_TEST_EVENTSHUB=${KNATIVE_EVENTING_TEST_EVENTSHUB:-"${eventing}/eventshub:${tag}"}
  export KNATIVE_EVENTING_TEST_PERFORMANCE=${KNATIVE_EVENTING_TEST_PERFORMANCE:-"${eventing}/performance:${tag}"}
  export KNATIVE_EVENTING_TEST_PRINT=${KNATIVE_EVENTING_TEST_PRINT:-"${eventing}/print:${tag}"}
  export KNATIVE_EVENTING_TEST_RECORDEVENTS=${KNATIVE_EVENTING_TEST_RECORDEVENTS:-"${eventing}/recordevents:${tag}"}
  export KNATIVE_EVENTING_TEST_REQUEST_SENDER=${KNATIVE_EVENTING_TEST_REQUEST_SENDER:-"${eventing}/request-sender:${tag}"}
  export KNATIVE_EVENTING_TEST_WATHOLA_FETCHER=${KNATIVE_EVENTING_TEST_WATHOLA_FETCHER:-"${eventing}/wathola-fetcher:${tag}"}
  export KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER=${KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER:-"${eventing}/wathola-forwarder:${tag}"}
  export KNATIVE_EVENTING_TEST_WATHOLA_RECEIVER=${KNATIVE_EVENTING_TEST_WATHOLA_RECEIVER:-"${eventing}/wathola-receiver:${tag}"}
  export KNATIVE_EVENTING_TEST_WATHOLA_SENDER=${KNATIVE_EVENTING_TEST_WATHOLA_SENDER:-"${eventing}/wathola-sender:${tag}"}
}

function default_knative_eventing_istio_images() {
  local eventing_istio
  eventing_istio="${registry}/knative-eventing-istio"
  local tag
  tag=$(metadata.get dependencies.eventing_istio)
  export KNATIVE_EVENTING_ISTIO_CONTROLLER=${KNATIVE_EVENTING_ISTIO_CONTROLLER:-"${eventing_istio}-controller:${tag}"}
}

function default_knative_eventing_kafka_broker_images() {
  local eventing_kafka_broker
  local tag
  tag=$(metadata.get dependencies.eventing_kafka_broker)
  eventing_kafka_broker="${registry}/knative-eventing-kafka-broker"
  export KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER=${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER:-"${eventing_kafka_broker}-dispatcher:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER=${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER:-"${eventing_kafka_broker}-receiver:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER=${KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER:-"${eventing_kafka_broker}-kafka-controller:${tag}"}
  export KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA=${KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA:-"${eventing_kafka_broker}-webhook-kafka":${tag}}
  export KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL=${KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL:-"${eventing_kafka_broker}-post-install:${tag}"}
}

function default_knative_ingress_images() {
  local knative_kourier knative_istio
  knative_kourier="$(metadata.get dependencies.kourier)"
  export KNATIVE_KOURIER_CONTROL=${KNATIVE_KOURIER_CONTROL:-"${registry}/net-kourier-kourier:${knative_kourier}"}
  export KNATIVE_KOURIER_GATEWAY=${KNATIVE_KOURIER_GATEWAY:-"quay.io/maistra-dev/proxyv2-ubi8:$(metadata.get dependencies.maistra)"}

  knative_istio="$(metadata.get dependencies.net_istio)"
  export KNATIVE_ISTIO_CONTROLLER=${KNATIVE_ISTIO_CONTROLLER:-"${registry}/net-istio-controller:${knative_istio}"}
  export KNATIVE_ISTIO_WEBHOOK=${KNATIVE_ISTIO_WEBHOOK:-"${registry}/net-istio-webhook:${knative_istio}"}
}
