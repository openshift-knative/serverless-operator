#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/metadata.bash"

ci_registry_host='registry.ci.openshift.org'
ci_registry="${ci_registry_host}/openshift"

export CURRENT_VERSION_IMAGES=${CURRENT_VERSION_IMAGES:-"main"}
CURRENT_VERSION="$(metadata.get project.version)"

quay_registry_app_version=${CURRENT_VERSION/./} # 1.34.0 -> 134.0
quay_registry_app_version=${quay_registry_app_version%.*} # 134.0 -> 134
registry_prefix="quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-"
registry_host="${registry_prefix}${quay_registry_app_version}"
registry="${registry_host}"
serverless_registry="${registry_host}/serverless"

function default_serverless_operator_images() {
  export SERVERLESS_KNATIVE_OPERATOR=${SERVERLESS_KNATIVE_OPERATOR:-$(latest_konflux_image_sha "${serverless_registry}-kn-operator:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR=${SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR:-$(latest_konflux_image_sha "${serverless_registry}-openshift-kn-operator:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_INGRESS=${SERVERLESS_INGRESS:-$(latest_konflux_image_sha "${serverless_registry}-ingress:${CURRENT_VERSION_IMAGES}")}

  export SERVERLESS_BUNDLE=${SERVERLESS_BUNDLE:-$(latest_konflux_image_sha "${serverless_registry}-bundle:${CURRENT_VERSION_IMAGES}")}
  export DEFAULT_SERVERLESS_BUNDLE=${DEFAULT_SERVERLESS_BUNDLE:-$(latest_konflux_image_sha "${serverless_registry}-bundle:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_INDEX=${SERVERLESS_INDEX:-$(latest_konflux_image_sha "${serverless_registry}-index:${CURRENT_VERSION_IMAGES}")}
}

function knative_serving_images_release() {
  knative_serving_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_serving_images() {
  knative_serving_images "$(metadata.get dependencies.serving)"
}

function knative_serving_images() {
  local serving tag app_version
  serving="${registry}/knative-serving"
  tag=${1:?"Provide tag for Serving images"}

  app_version=$(get_app_version_from_tag "${tag}")
  serving="${registry_prefix}${app_version}/kn-serving"

  export KNATIVE_SERVING_QUEUE=${KNATIVE_SERVING_QUEUE:-$(latest_konflux_image_sha "${serving}-queue:${tag}")}
  export KNATIVE_SERVING_ACTIVATOR=${KNATIVE_SERVING_ACTIVATOR:-$(latest_konflux_image_sha "${serving}-activator:${tag}")}
  export KNATIVE_SERVING_AUTOSCALER=${KNATIVE_SERVING_AUTOSCALER:-$(latest_konflux_image_sha "${serving}-autoscaler:${tag}")}
  export KNATIVE_SERVING_AUTOSCALER_HPA=${KNATIVE_SERVING_AUTOSCALER_HPA:-$(latest_konflux_image_sha "${serving}-autoscaler-hpa:${tag}")}
  export KNATIVE_SERVING_CONTROLLER=${KNATIVE_SERVING_CONTROLLER:-$(latest_konflux_image_sha "${serving}-controller:${tag}")}
  export KNATIVE_SERVING_WEBHOOK=${KNATIVE_SERVING_WEBHOOK:-$(latest_konflux_image_sha "${serving}-webhook:${tag}")}
  export KNATIVE_SERVING_STORAGE_VERSION_MIGRATION=${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION:-$(latest_konflux_image_sha "${serving}-storage-version-migration:${tag}")}
}

function knative_eventing_images_release() {
  knative_eventing_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_eventing_images() {
  knative_eventing_images "$(metadata.get dependencies.eventing)"
}

function knative_eventing_images() {
  local eventing tag app_version
  tag=${1:?"Provide tag for Eventing images"}

  app_version=$(get_app_version_from_tag "${tag}")
  eventing="${registry_prefix}${app_version}/kn-eventing"

  export KNATIVE_EVENTING_CONTROLLER=${KNATIVE_EVENTING_CONTROLLER:-$(latest_konflux_image_sha "${eventing}-controller:${tag}")}
  export KNATIVE_EVENTING_WEBHOOK=${KNATIVE_EVENTING_WEBHOOK:-$(latest_konflux_image_sha "${eventing}-webhook:${tag}")}
  export KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION=${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION:-$(latest_konflux_image_sha "${eventing}-migrate:${tag}")}
  export KNATIVE_EVENTING_INGRESS=${KNATIVE_EVENTING_INGRESS:-$(latest_konflux_image_sha "${eventing}-ingress:${tag}")}
  export KNATIVE_EVENTING_FILTER=${KNATIVE_EVENTING_FILTER:-$(latest_konflux_image_sha "${eventing}-filter:${tag}")}
  export KNATIVE_EVENTING_MTCHANNEL_BROKER=${KNATIVE_EVENTING_MTCHANNEL_BROKER:-$(latest_konflux_image_sha "${eventing}-mtchannel-broker:${tag}")}
  export KNATIVE_EVENTING_MTPING=${KNATIVE_EVENTING_MTPING:-$(latest_konflux_image_sha "${eventing}-mtping:${tag}")}
  export KNATIVE_EVENTING_CHANNEL_CONTROLLER=${KNATIVE_EVENTING_CHANNEL_CONTROLLER:-$(latest_konflux_image_sha "${eventing}-channel-controller:${tag}")}
  export KNATIVE_EVENTING_CHANNEL_DISPATCHER=${KNATIVE_EVENTING_CHANNEL_DISPATCHER:-$(latest_konflux_image_sha "${eventing}-channel-dispatcher:${tag}")}
  export KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER=${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER:-$(latest_konflux_image_sha "${eventing}-apiserver-receive-adapter:${tag}")}
  export KNATIVE_EVENTING_JOBSINK=${KNATIVE_EVENTING_JOBSINK:-$(latest_konflux_image_sha "${eventing}-jobsink:${tag}")}

  export KNATIVE_EVENTING_APPENDER=${KNATIVE_EVENTING_APPENDER:-$(latest_konflux_image_sha "${eventing}-appender:${tag}")}
  export KNATIVE_EVENTING_EVENT_DISPLAY=${KNATIVE_EVENTING_EVENT_DISPLAY:-$(latest_konflux_image_sha "${eventing}-event-display:${tag}")}
  export KNATIVE_EVENTING_HEARTBEATS_RECEIVER=${KNATIVE_EVENTING_HEARTBEATS_RECEIVER:-$(latest_konflux_image_sha "${eventing}-heartbeats-receiver:${tag}")}
  export KNATIVE_EVENTING_HEARTBEATS=${KNATIVE_EVENTING_HEARTBEATS:-$(latest_konflux_image_sha "${eventing}-heartbeats:${tag}")}
  export KNATIVE_EVENTING_MIGRATE=${KNATIVE_EVENTING_MIGRATE:-$(latest_konflux_image_sha "${eventing}-migrate:${tag}")}
  export KNATIVE_EVENTING_PONG=${KNATIVE_EVENTING_PONG:-$(latest_konflux_image_sha "${eventing}-pong:${tag}")}
  export KNATIVE_EVENTING_SCHEMA=${KNATIVE_EVENTING_SCHEMA:-$(latest_konflux_image_sha "${eventing}-schema:${tag}")}

  # Test images
  local eventing_test="${eventing}-test"
  export KNATIVE_EVENTING_TEST_EVENT_SENDER=${KNATIVE_EVENTING_TEST_EVENT_SENDER:-$(latest_konflux_image_sha "${eventing_test}-event-sender:${tag}")}
  export KNATIVE_EVENTING_TEST_EVENTSHUB=${KNATIVE_EVENTING_TEST_EVENTSHUB:-$(latest_konflux_image_sha "${eventing_test}-eventshub:${tag}")}
  export KNATIVE_EVENTING_TEST_PRINT=${KNATIVE_EVENTING_TEST_PRINT:-$(latest_konflux_image_sha "${eventing_test}-print:${tag}")}
  export KNATIVE_EVENTING_TEST_RECORDEVENTS=${KNATIVE_EVENTING_TEST_RECORDEVENTS:-$(latest_konflux_image_sha "${eventing_test}-recordevents:${tag}")}
  export KNATIVE_EVENTING_TEST_REQUEST_SENDER=${KNATIVE_EVENTING_TEST_REQUEST_SENDER:-$(latest_konflux_image_sha "${eventing_test}-request-sender:${tag}")}
  export KNATIVE_EVENTING_TEST_WATHOLA_FETCHER=${KNATIVE_EVENTING_TEST_WATHOLA_FETCHER:-$(latest_konflux_image_sha "${eventing_test}-wathola-fetcher:${tag}")}
  export KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER=${KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER:-$(latest_konflux_image_sha "${eventing_test}-wathola-forwarder:${tag}")}
  export KNATIVE_EVENTING_TEST_WATHOLA_RECEIVER=${KNATIVE_EVENTING_TEST_WATHOLA_RECEIVER:-$(latest_konflux_image_sha "${eventing_test}-wathola-receiver:${tag}")}
  export KNATIVE_EVENTING_TEST_WATHOLA_SENDER=${KNATIVE_EVENTING_TEST_WATHOLA_SENDER:-$(latest_konflux_image_sha "${eventing_test}-wathola-sender:${tag}")}
}

function knative_eventing_istio_images_release() {
  knative_eventing_istio_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_eventing_istio_images() {
  knative_eventing_istio_images "$(metadata.get dependencies.eventing_istio)"
}

function knative_eventing_istio_images() {
  local eventing_istio tag app_version
  tag=${1:?"Provide tag for Eventing Istio images"}

  app_version=$(get_app_version_from_tag "${tag}")
  eventing_istio="${registry_prefix}${app_version}/kn-eventing-istio"

  export KNATIVE_EVENTING_ISTIO_CONTROLLER=${KNATIVE_EVENTING_ISTIO_CONTROLLER:-"${eventing_istio}-controller:${tag}"}
}

function knative_eventing_kafka_broker_images_release() {
  knative_eventing_kafka_broker_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_eventing_kafka_broker_images() {
  knative_eventing_kafka_broker_images "$(metadata.get dependencies.eventing_kafka_broker)"
}

function knative_eventing_kafka_broker_images() {
  local eventing_kafka_broker tag app_version
  tag=${1:?"Provide tag for Eventing Kafka Broker images"}

  app_version=$(get_app_version_from_tag "${tag}")
  eventing_kafka_broker="${registry_prefix}${app_version}/kn-ekb"

  export KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER=${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER:-$(latest_konflux_image_sha "${eventing_kafka_broker}-dispatcher:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER=${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER:-$(latest_konflux_image_sha "${eventing_kafka_broker}-receiver:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER=${KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER:-$(latest_konflux_image_sha "${eventing_kafka_broker}-kafka-controller:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA=${KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA:-$(latest_konflux_image_sha "${eventing_kafka_broker}-webhook-kafka":${tag})}
  export KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL=${KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL:-$(latest_konflux_image_sha "${eventing_kafka_broker}-post-install:${tag}")}
}

function default_knative_ingress_images() {
  local kourier_registry istio_registry knative_kourier knative_istio kourier_app_version istio_app_version

  knative_kourier="$(metadata.get dependencies.kourier)"
  kourier_app_version=$(get_app_version_from_tag "${knative_kourier}")
  kourier_registry="${registry_prefix}${kourier_app_version}/net-kourier"

  export KNATIVE_KOURIER_CONTROL=${KNATIVE_KOURIER_CONTROL:-$(latest_konflux_image_sha "${kourier_registry}-kourier:${knative_kourier}")}
  export KNATIVE_KOURIER_GATEWAY=${KNATIVE_KOURIER_GATEWAY:-"quay.io/maistra-dev/proxyv2-ubi8:$(metadata.get dependencies.maistra)"}

  knative_istio="$(metadata.get dependencies.net_istio)"
  istio_app_version=$(get_app_version_from_tag "${knative_istio}")
  istio_registry="${registry_prefix}${istio_app_version}/net-istio"

  export KNATIVE_ISTIO_CONTROLLER=${KNATIVE_ISTIO_CONTROLLER:-$(latest_konflux_image_sha "${istio_registry}-controller:${knative_istio}")}
  export KNATIVE_ISTIO_WEBHOOK=${KNATIVE_ISTIO_WEBHOOK:-$(latest_konflux_image_sha "${istio_registry}-webhook:${knative_istio}")}
}

function knative_backstage_plugins_images() {
  local backstage_plugins tag
  # TODO migrate to Konflux
  backstage_plugins="${ci_registry}/knative-backstage-plugins"
  tag=${1:?"Provide tag for Backstage plugins images"}
  export KNATIVE_BACKSTAGE_PLUGINS_EVENTMESH=${KNATIVE_BACKSTAGE_PLUGINS_EVENTMESH:-"${backstage_plugins}-eventmesh:${tag}"}
}

function knative_backstage_plugins_images_release() {
  knative_backstage_plugins_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_backstage_plugins_images() {
  knative_backstage_plugins_images "$(metadata.get dependencies.backstage_plugins)"
}

function latest_konflux_image_sha() {
  input=${1:?"Provide image"}

  image_without_tag=${input%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  go_bin="$(go env GOPATH)/bin"
  export GOPATH="$PATH:$go_bin"
  digest=$(skopeo inspect "docker://${image_without_tag}:latest" | jq -r '.Digest')
  if [ "${digest}" = "" ]; then
    exit 1
  fi

  echo "${image_without_tag}@${digest}"
}

function get_app_version_from_tag() {
  local tag app_version
  tag=${1:?"Provide tag for Serving images"}

  app_version=$(sobranch --upstream-version "${tag/knative-v/}") # -> release-1.34
  app_version=${app_version/release-/}                   # -> 1.34
  app_version=${app_version/./}                          # -> 134
  echo "${app_version}"
}
