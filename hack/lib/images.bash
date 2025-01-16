#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/metadata.bash"

ci_registry_host='registry.ci.openshift.org'
ci_registry="${ci_registry_host}/openshift"

export CURRENT_VERSION_IMAGES=${CURRENT_VERSION_IMAGES:-"main"}
CURRENT_VERSION="$(metadata.get project.version)"

quay_registry_app_version=${CURRENT_VERSION/./} # 1.34.0 -> 134.0
quay_registry_app_version=${quay_registry_app_version%.*} # 134.0 -> 134
registry_prefix_quay="quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-"
registry_quay="${registry_prefix_quay}${quay_registry_app_version}"
registry_redhat_io="registry.redhat.io/openshift-serverless-1"

# Add extra flags to skopeo cmd, in case of MacOS add OS/arch overrides
export SKOPEO_EXTRA_FLAGS="${SKOPE_EXTRA_FLAGS:-}"
if [[ "$(uname -s)" == "Darwin" ]]; then
  SKOPEO_EXTRA_FLAGS="${SKOPEO_EXTRA_FLAGS} --override-os linux --override-arch amd64"
fi

function get_serverless_operator_rhel_version() {
  sorhel --so-version="${CURRENT_VERSION}"
}

function default_serverless_operator_images() {
  local ocp_version
  local serverless_registry="${registry_quay}/serverless"

  export SERVERLESS_KNATIVE_OPERATOR=${SERVERLESS_KNATIVE_OPERATOR:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-kn-operator:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR=${SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-openshift-kn-operator:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_INGRESS=${SERVERLESS_INGRESS:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-ingress:${CURRENT_VERSION_IMAGES}")}
  export SERVERLESS_MUST_GATHER=${SERVERLESS_MUST_GATHER:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-must-gather:${CURRENT_VERSION_IMAGES}")}

  export SERVERLESS_BUNDLE=${SERVERLESS_BUNDLE:-$(get_bundle_for_version "${CURRENT_VERSION}")}
  export DEFAULT_SERVERLESS_BUNDLE=${DEFAULT_SERVERLESS_BUNDLE:-$(get_bundle_for_version "${CURRENT_VERSION}")}

  SERVERLESS_BUNDLE_REDHAT_IO=${SERVERLESS_BUNDLE_REDHAT_IO:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-bundle:${CURRENT_VERSION_IMAGES}")}
  # Bundle image is in different locations in quay.io and registry.redhat.io
  export SERVERLESS_BUNDLE_REDHAT_IO=${SERVERLESS_BUNDLE_REDHAT_IO//serverless-bundle/serverless-operator-bundle}

  # Use the current OCP version if the cluster is running otherwise use the latest.
  if oc get clusterversion &>/dev/null; then
    ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
    ocp_version=$(versions.major_minor "$ocp_version")
  else
    ocp_version=$(metadata.get 'requirements.ocpVersion.min')
  fi
  ocp_version=${ocp_version/./} # 4.17 -> 417

  export INDEX_IMAGE=${INDEX_IMAGE:-$(latest_konflux_image_sha "${registry_quay}-fbc-${ocp_version}/serverless-index-${quay_registry_app_version}-fbc-${ocp_version}:${CURRENT_VERSION_IMAGES}")}
}

# Bundle image is specific as we need to pull older versions for including in the catalog.
function get_bundle_for_version() {
  local version app_version bundle
  version=${1:?"Provide version for Bundle image"}

  app_version=${version/./} # 1.34.0 -> 134.0
  app_version=${app_version%.*} # 134.0 -> 134

  bundle="${registry_prefix_quay}${app_version}/serverless-bundle"

  image=$(image_with_sha "${bundle}:latest")
  # As a backup, try also CI registry. This it temporary until the previous version gets to Konflux.
  if [[ "${image}" == "" ]]; then
    image=$(image_with_sha "registry.ci.openshift.org/knative/serverless-bundle:release-${version}")
  fi

  if [[ "${image}" == "" ]]; then
    exit 1
  fi

  echo "$image"
}

function knative_serving_images_release() {
  knative_serving_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_serving_images() {
  knative_serving_images "$(metadata.get dependencies.serving)"
}

function knative_serving_images() {
  local serving tag app_version
  tag=${1:?"Provide tag for Serving images"}

  app_version=$(get_app_version_from_tag "${tag}")
  serving="${registry_prefix_quay}${app_version}/kn-serving"

  export KNATIVE_SERVING_QUEUE=${KNATIVE_SERVING_QUEUE:-$(latest_registry_redhat_io_image_sha "${serving}-queue:${tag}")}
  export KNATIVE_SERVING_ACTIVATOR=${KNATIVE_SERVING_ACTIVATOR:-$(latest_registry_redhat_io_image_sha "${serving}-activator:${tag}")}
  export KNATIVE_SERVING_AUTOSCALER=${KNATIVE_SERVING_AUTOSCALER:-$(latest_registry_redhat_io_image_sha "${serving}-autoscaler:${tag}")}
  export KNATIVE_SERVING_AUTOSCALER_HPA=${KNATIVE_SERVING_AUTOSCALER_HPA:-$(latest_registry_redhat_io_image_sha "${serving}-autoscaler-hpa:${tag}")}
  export KNATIVE_SERVING_CONTROLLER=${KNATIVE_SERVING_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${serving}-controller:${tag}")}
  export KNATIVE_SERVING_WEBHOOK=${KNATIVE_SERVING_WEBHOOK:-$(latest_registry_redhat_io_image_sha "${serving}-webhook:${tag}")}
  export KNATIVE_SERVING_STORAGE_VERSION_MIGRATION=${KNATIVE_SERVING_STORAGE_VERSION_MIGRATION:-$(latest_registry_redhat_io_image_sha "${serving}-storage-version-migration:${tag}")}

  export KNATIVE_SERVING_IMAGE_PREFIX="${serving}"
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
  eventing="${registry_prefix_quay}${app_version}/kn-eventing"

  export KNATIVE_EVENTING_CONTROLLER=${KNATIVE_EVENTING_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${eventing}-controller:${tag}")}
  export KNATIVE_EVENTING_WEBHOOK=${KNATIVE_EVENTING_WEBHOOK:-$(latest_registry_redhat_io_image_sha "${eventing}-webhook:${tag}")}
  export KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION=${KNATIVE_EVENTING_STORAGE_VERSION_MIGRATION:-$(latest_registry_redhat_io_image_sha "${eventing}-migrate:${tag}")}
  export KNATIVE_EVENTING_INGRESS=${KNATIVE_EVENTING_INGRESS:-$(latest_registry_redhat_io_image_sha "${eventing}-ingress:${tag}")}
  export KNATIVE_EVENTING_FILTER=${KNATIVE_EVENTING_FILTER:-$(latest_registry_redhat_io_image_sha "${eventing}-filter:${tag}")}
  export KNATIVE_EVENTING_MTCHANNEL_BROKER=${KNATIVE_EVENTING_MTCHANNEL_BROKER:-$(latest_registry_redhat_io_image_sha "${eventing}-mtchannel-broker:${tag}")}
  export KNATIVE_EVENTING_MTPING=${KNATIVE_EVENTING_MTPING:-$(latest_registry_redhat_io_image_sha "${eventing}-mtping:${tag}")}
  export KNATIVE_EVENTING_CHANNEL_CONTROLLER=${KNATIVE_EVENTING_CHANNEL_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${eventing}-channel-controller:${tag}")}
  export KNATIVE_EVENTING_CHANNEL_DISPATCHER=${KNATIVE_EVENTING_CHANNEL_DISPATCHER:-$(latest_registry_redhat_io_image_sha "${eventing}-channel-dispatcher:${tag}")}
  export KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER=${KNATIVE_EVENTING_APISERVER_RECEIVE_ADAPTER:-$(latest_registry_redhat_io_image_sha "${eventing}-apiserver-receive-adapter:${tag}")}
  export KNATIVE_EVENTING_JOBSINK=${KNATIVE_EVENTING_JOBSINK:-$(latest_registry_redhat_io_image_sha "${eventing}-jobsink:${tag}")}
  export KNATIVE_EVENTING_MIGRATE=${KNATIVE_EVENTING_MIGRATE:-$(latest_registry_redhat_io_image_sha "${eventing}-migrate:${tag}")}

  # Point to Konflux Quay repo for images not present in ClusterServiceVersion.
  export KNATIVE_EVENTING_APPENDER=${KNATIVE_EVENTING_APPENDER:-$(latest_konflux_image_sha "${eventing}-appender:${tag}")}
  export KNATIVE_EVENTING_EVENT_DISPLAY=${KNATIVE_EVENTING_EVENT_DISPLAY:-$(latest_konflux_image_sha "${eventing}-event-display:${tag}")}
  export KNATIVE_EVENTING_HEARTBEATS_RECEIVER=${KNATIVE_EVENTING_HEARTBEATS_RECEIVER:-$(latest_konflux_image_sha "${eventing}-heartbeats-receiver:${tag}")}
  export KNATIVE_EVENTING_HEARTBEATS=${KNATIVE_EVENTING_HEARTBEATS:-$(latest_konflux_image_sha "${eventing}-heartbeats:${tag}")}
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
  eventing_istio="${registry_prefix_quay}${app_version}/kn-eventing-istio"

  export KNATIVE_EVENTING_ISTIO_CONTROLLER=${KNATIVE_EVENTING_ISTIO_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${eventing_istio}-controller:${tag}")}
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
  eventing_kafka_broker="${registry_prefix_quay}${app_version}/kn-ekb"

  export KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER=${KNATIVE_EVENTING_KAFKA_BROKER_DISPATCHER:-$(latest_registry_redhat_io_image_sha "${eventing_kafka_broker}-dispatcher:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER=${KNATIVE_EVENTING_KAFKA_BROKER_RECEIVER:-$(latest_registry_redhat_io_image_sha "${eventing_kafka_broker}-receiver:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER=${KNATIVE_EVENTING_KAFKA_BROKER_KAFKA_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${eventing_kafka_broker}-kafka-controller:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA=${KNATIVE_EVENTING_KAFKA_BROKER_WEBHOOK_KAFKA:-$(latest_registry_redhat_io_image_sha "${eventing_kafka_broker}-webhook-kafka:${tag}")}
  export KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL=${KNATIVE_EVENTING_KAFKA_BROKER_POST_INSTALL:-$(latest_registry_redhat_io_image_sha "${eventing_kafka_broker}-post-install:${tag}")}

  export KNATIVE_EVENTING_KAFKA_BROKER_TEST_KAFKA_CONSUMER=${KNATIVE_EVENTING_KAFKA_BROKER_TEST_KAFKA_CONSUMER:-$(latest_konflux_image_sha "${eventing_kafka_broker}-test-kafka-consumer")}
}

function knative_kn_plugin_func_images_release() {
  knative_kn_plugin_func_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_kn_plugin_func_images() {
  knative_kn_plugin_func_images "$(metadata.get dependencies.func.promotion_tag)"
}

function knative_kn_plugin_func_images() {
  local knative_kn_plugin_func tag app_version
  tag=${1:?"Provide tag for kn-plugin-func images"}

  app_version=$(get_app_version_from_tag "${tag}")
  knative_kn_plugin_func="${registry_prefix_quay}${app_version}/kn-plugin-func"

  export KNATIVE_KN_PLUGIN_FUNC_FUNC_UTIL=${KNATIVE_KN_PLUGIN_FUNC_FUNC_UTIL:-$(latest_registry_redhat_io_image_sha "${knative_kn_plugin_func}-func-util:${tag}")}

  export KNATIVE_KN_PLUGIN_FUNC_TEKTON_S2I=${KNATIVE_KN_PLUGIN_FUNC_UTIL:-"$(metadata.get dependencies.func.tekton_s2i)"}
  export KNATIVE_KN_PLUGIN_FUNC_TEKTON_BUILDAH=${KNATIVE_KN_PLUGIN_FUNC_UTIL:-"$(metadata.get dependencies.func.tekton_buildah)"}
  export KNATIVE_KN_PLUGIN_FUNC_NODEJS_20_MINIMAL=${KNATIVE_KN_PLUGIN_FUNC_UTIL:-"$(metadata.get dependencies.func.nodejs_20_minimal)"}
  export KNATIVE_KN_PLUGIN_FUNC_OPENJDK_21=${KNATIVE_KN_PLUGIN_FUNC_UTIL:-"$(metadata.get dependencies.func.openjdk_21)"}
  export KNATIVE_KN_PLUGIN_FUNC_PYTHON_39=${KNATIVE_KN_PLUGIN_FUNC_UTIL:-"$(metadata.get dependencies.func.python-39)"}
}

function knative_kn_plugin_event_images_release() {
  knative_kn_plugin_event_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_kn_plugin_event_images() {
  knative_kn_plugin_event_images "$(metadata.get dependencies.cli)"
}

function knative_kn_plugin_event_images() {
  local knative_kn_plugin_event tag app_version
  tag=${1:?"Provide tag for kn-plugin-event images"}

  app_version=$(get_app_version_from_tag "${tag}")
  knative_kn_plugin_event="${registry_prefix_quay}${app_version}/kn-plugin-event"

  export KNATIVE_KN_PLUGIN_EVENT_SENDER=${KNATIVE_KN_PLUGIN_EVENT_SENDER:-$(latest_registry_redhat_io_image_sha "${knative_kn_plugin_event}-sender:${tag}")}
}

function knative_client_images_release() {
  knative_client_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_client_images() {
  knative_client_images "$(metadata.get dependencies.cli)"
}

function knative_client_images() {
  local knative_client tag app_version
  tag=${1:?"Provide tag for kn-client images"}

  app_version=$(get_app_version_from_tag "${tag}")
  knative_client="${registry_prefix_quay}${app_version}/kn-client"

  export KNATIVE_KN_CLIENT=${KNATIVE_KN_CLIENT:-$(latest_registry_redhat_io_image_sha "${knative_client}-kn:${tag}")}
  export KNATIVE_KN_CLIENT_CLI_ARTIFACTS=${KNATIVE_KN_CLIENT_CLI_ARTIFACTS:-$(latest_registry_redhat_io_image_sha "${knative_client}-cli-artifacts:${tag}")}
}

function default_knative_ingress_images() {
  local kourier_registry istio_registry knative_kourier knative_istio kourier_app_version istio_app_version

  knative_kourier="$(metadata.get dependencies.kourier)"
  kourier_app_version=$(get_app_version_from_tag "${knative_kourier}")
  kourier_registry="${registry_prefix_quay}${kourier_app_version}/net-kourier"

  export KNATIVE_KOURIER_CONTROL=${KNATIVE_KOURIER_CONTROL:-$(latest_registry_redhat_io_image_sha "${kourier_registry}-kourier:${knative_kourier}")}
  export KNATIVE_KOURIER_GATEWAY=${KNATIVE_KOURIER_GATEWAY:-"$(metadata.get dependencies.service_mesh_proxy)"}

  knative_istio="$(metadata.get dependencies.net_istio)"
  istio_app_version=$(get_app_version_from_tag "${knative_istio}")
  istio_registry="${registry_prefix_quay}${istio_app_version}/net-istio"

  export KNATIVE_ISTIO_CONTROLLER=${KNATIVE_ISTIO_CONTROLLER:-$(latest_registry_redhat_io_image_sha "${istio_registry}-controller:${knative_istio}")}
  export KNATIVE_ISTIO_WEBHOOK=${KNATIVE_ISTIO_WEBHOOK:-$(latest_registry_redhat_io_image_sha "${istio_registry}-webhook:${knative_istio}")}
}

function knative_backstage_plugins_images() {
  local backstage_plugins tag app_version
  tag=${1:?"Provide tag for Backstage plugins images"}

  app_version=$(get_app_version_from_tag "${tag}")
  backstage_plugins="${registry_prefix_quay}${app_version}/kn-backstage-plugins"

  export KNATIVE_BACKSTAGE_PLUGINS_EVENTMESH=${KNATIVE_BACKSTAGE_PLUGINS_EVENTMESH:-$(latest_registry_redhat_io_image_sha "${backstage_plugins}-eventmesh:${tag}")}
}

function knative_backstage_plugins_images_release() {
  knative_backstage_plugins_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_backstage_plugins_images() {
  knative_backstage_plugins_images "$(metadata.get dependencies.backstage_plugins)"
}

function latest_registry_redhat_io_image_sha() {
  input=${1:?"Provide image"}

  image_without_tag=${input%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  image=$(image_with_sha "${image_without_tag}:latest")

  if [ "${image}" = "" ]; then
    exit 1
  fi

  rh_registry_image="$(get_rh_registry_image_ref "$image")"
  echo "$rh_registry_image"
}

function latest_konflux_image_sha() {
  input=${1:?"Provide image"}
  tag=${2:-"latest"}

  image_without_tag=${input%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  image=$(image_with_sha "${image_without_tag}:${tag}")

  if [ "${image}" = "" ]; then
    exit 1
  fi

  echo "${image}"
}

function image_with_sha {
  image=${1:?"Provide image"}

  # shellcheck disable=SC2086
  digest=$(skopeo inspect --no-tags=true ${SKOPEO_EXTRA_FLAGS} "docker://${image}" | jq -r '.Digest')
  if [ "${digest}" = "" ]; then
    echo ""
  fi

  image_without_tag=${image%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

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

# returns the quay image for a given rh registry image ref
function get_quay_image_ref() {
  local rh_registry_image_ref
  rh_registry_image_ref="${1}"

  if  [[ $rh_registry_image_ref =~ $registry_redhat_io ]]; then
    image=${rh_registry_image_ref##*/} # Get image name after last slash
    image_sha=${image##*@} # Get SHA of image
    image_name=${image%@*} # Remove sha

    if [[ "${image_name}" =~ ^serverless-openshift-kn-rhel[0-9]+-operator$ ]]; then
      # serverless-openshift-kn-operator is special, as it has rhel in the middle of the name
      # see https://redhat-internal.slack.com/archives/CKR568L8G/p1729684088850349
      component="serverless-openshift-kn-operator"
    elif [[ "${image_name}" == "serverless-operator-bundle" ]]; then
      # serverless-operator-bundle is special, as it is named only serverless-bundle in quay
      component="serverless-bundle"
    else
      # for other images simply remove the -rhelXYZ suffix
      component=${image_name%-rhel*}
    fi

    echo "${registry_quay}/${component}@${image_sha}"
  else
    echo "Image must be from ${registry_redhat_io}, got ${rh_registry_image_ref}"
    return 1
  fi
}

# returns the RH registry image for a given quay image ref
function get_rh_registry_image_ref() {
  local quay_registry_image_ref
  quay_registry_image_ref="${1}"

  if  [[ $quay_registry_image_ref =~ $registry_quay ]]; then
    image=${quay_registry_image_ref##*/} # Get image name after last slash
    image_sha=${image##*@} # Get SHA of image
    image_name=${image%@*} # Remove sha

    # Add rhel suffix
    if [ "${image_name}" == "serverless-openshift-kn-operator" ]; then
      # serverless-openshift-kn-operator is special, as it has rhel in the middle of the name
      # see https://redhat-internal.slack.com/archives/CKR568L8G/p1729684088850349
      image_name="serverless-openshift-kn-rhel$(get_serverless_operator_rhel_version)-operator"
    elif [ "${image_name}" == "serverless-bundle" ]; then
      # serverless-bundle is special, as it has no rhelXYZ in the name
      image_name="serverless-bundle"
    else
      # for other images simply add it as a suffix
      image_name="${image_name}-rhel$(get_serverless_operator_rhel_version)"
    fi

    echo "${registry_redhat_io}/${image_name}@${image_sha}"
  else
    echo "Image must be from ${registry_quay}, got ${quay_registry_image_ref}"
    return 1
  fi
}
