#!/usr/bin/env bash

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/metadata.bash"

ci_registry_host='registry.ci.openshift.org'
ci_registry="${ci_registry_host}/openshift"

export CURRENT_VERSION_IMAGES=${CURRENT_VERSION_IMAGES:-"main"}
CURRENT_VERSION="$(metadata.get project.version)"
PREVIOUS_VERSION="$(metadata.get olm.replaces)"

registry_prefix_quay="quay.io/redhat-user-workloads/ocp-serverless-tenant/serverless-operator-"

quay_registry_app_version=${CURRENT_VERSION/./} # 1.36.0 -> 136.0
quay_registry_app_version=${quay_registry_app_version%.*} # 136.0 -> 136

quay_registry_app_version_previous=${PREVIOUS_VERSION/./} # 1.35.0 -> 135.0
quay_registry_app_version_previous=${quay_registry_app_version_previous%.*} # 135.0 -> 135

quay_registry_app_version_next=${CURRENT_VERSION/./} # 1.36.0 -> 136.0
quay_registry_app_version_next=${quay_registry_app_version_next%.*} # 136.0 -> 136
quay_registry_app_version_next="$((quay_registry_app_version_next + 1))" # 136 -> 137

export registry_quay="${registry_prefix_quay}${quay_registry_app_version}"
export registry_quay_previous="${registry_prefix_quay}${quay_registry_app_version_previous}"
export registry_quay_next="${registry_prefix_quay}${quay_registry_app_version_next}"
export registry_redhat_io="registry.redhat.io/openshift-serverless-1"

export FORCE_USE_QUAY_IMAGES=${FORCE_USE_QUAY_IMAGES:-"false"}

# Add extra flags to skopeo cmd, in case of MacOS add OS/arch overrides
export SKOPEO_EXTRA_FLAGS="${SKOPE_EXTRA_FLAGS:-}"
if [[ "$(uname -s)" == "Darwin" ]]; then
  SKOPEO_EXTRA_FLAGS="${SKOPEO_EXTRA_FLAGS} --override-os linux --override-arch amd64"
fi

function get_serverless_operator_rhel_version() {
  # Memoized via an exported variable (see images.init_inspect_cache), as this is called once per
  # image from command substitutions, where a plain assignment would not persist.
  if [[ -n "${SERVERLESS_OPERATOR_RHEL_VERSION:-}" ]]; then
    echo "${SERVERLESS_OPERATOR_RHEL_VERSION}"
    return
  fi
  sorhel --so-version="${CURRENT_VERSION}"
}

# The resolver functions needed to generate the ClusterServiceVersion (and, as a subset, all other
# release files). Honors USE_RELEASE_NEXT.
function images.csv_resolvers() {
  echo default_serverless_operator_images
  echo default_knative_ingress_images
  if [[ ${USE_RELEASE_NEXT:-} == "true" ]]; then
    echo knative_eventing_images_release
    echo knative_eventing_istio_images_release
    echo knative_eventing_kafka_broker_images_release
    echo knative_eventing_integrations_images_release
    echo knative_serving_images_release
    echo knative_kn_plugin_func_images_release
    echo knative_client_images_release
    echo knative_kn_plugin_event_images_release
  else
    echo default_knative_eventing_images
    echo default_knative_eventing_istio_images
    echo default_knative_eventing_kafka_broker_images
    echo default_knative_eventing_integrations_images
    echo default_knative_serving_images
    echo default_knative_kn_plugin_func_images
    echo default_knative_client_images
    echo default_knative_kn_plugin_event_images
  fi
}

# Sets up a directory caching `skopeo inspect` results, shared with child processes via the
# exported IMAGE_INSPECT_CACHE_DIR, so that every image is inspected at most once. The directory
# is removed when the calling script exits. No-op if a cache is already set up (e.g. by a parent).
function images.init_inspect_cache() {
  if [[ -z "${IMAGE_INSPECT_CACHE_DIR:-}" ]]; then
    IMAGE_INSPECT_CACHE_DIR="$(mktemp -d -t image-inspect-XXXXXX)"
    export IMAGE_INSPECT_CACHE_DIR
    # shellcheck disable=SC2064
    trap "rm -rf '${IMAGE_INSPECT_CACHE_DIR}'" EXIT
  fi
  SERVERLESS_OPERATOR_RHEL_VERSION="$(get_serverless_operator_rhel_version)"
  export SERVERLESS_OPERATOR_RHEL_VERSION
}

# Inspects, in parallel, all images the given resolver functions would look up and stores the
# results in the inspect cache, so that the subsequent (sequential) resolution is served from
# the cache. Requires images.init_inspect_cache.
function images.prefetch() {
  local images_list
  images_list="$(mktemp -t image-list-XXXXXX)"
  # Dry-run the resolvers in a subshell to collect the image references they look up, without
  # persisting any of the variables they export. Already exported variables are skipped by the
  # resolvers, so they don't contribute any image.
  # shellcheck disable=SC2030
  (
    export IMAGE_INSPECT_COLLECT_FILE="${images_list}"
    for resolver in "$@"; do
      "${resolver}"
    done
  ) > /dev/null 2>&1 || true

  export -f image_inspect image_inspect_cache_file
  # shellcheck disable=SC2016
  sort -u "${images_list}" | \
    xargs -P "${IMAGE_INSPECT_PARALLELISM:-16}" -I{} bash -c 'image_inspect "$1" > /dev/null 2>&1 || true' _ {}
  rm -f "${images_list}"
}

function default_serverless_operator_images() {
  local ocp_version
  local serverless_registry="${registry_quay}/serverless"

  # When we bump the metadata for a major version, images are not yet available for SO components on
  # the Konflux registry, so we use the `:latest` tag temporarily by passing "true" as argument to
  # various `latest_*` functions.
  #
  # Eventually, once images are available, SHAs will be replaced/used.

  export SERVERLESS_KNATIVE_OPERATOR=${SERVERLESS_KNATIVE_OPERATOR:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-kn-operator:${CURRENT_VERSION_IMAGES}" "true")}
  export SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR=${SERVERLESS_OPENSHIFT_KNATIVE_OPERATOR:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-openshift-kn-operator:${CURRENT_VERSION_IMAGES}" "true")}
  export SERVERLESS_INGRESS=${SERVERLESS_INGRESS:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-ingress:${CURRENT_VERSION_IMAGES}" "true")}
  export SERVERLESS_MUST_GATHER=${SERVERLESS_MUST_GATHER:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-must-gather:${CURRENT_VERSION_IMAGES}" "true")}

  # Differentiate between the bundle living on the Konflux quay registry (which is the actually
  # pullable URL) and the "final" bundle image URL eventually getting on registry.redhat.io.
  #
  # The actually pullable URL is used to generate the index images for FBC components and it
  # has to be (eventually) pullable.

  export DEFAULT_SERVERLESS_BUNDLE=${DEFAULT_SERVERLESS_BUNDLE:-$(get_bundle_for_version "${CURRENT_VERSION}")}
  export SERVERLESS_BUNDLE=${SERVERLESS_BUNDLE:-${DEFAULT_SERVERLESS_BUNDLE}}

  SERVERLESS_BUNDLE_REDHAT_IO=${SERVERLESS_BUNDLE_REDHAT_IO:-$(latest_registry_redhat_io_image_sha "${serverless_registry}-bundle:${CURRENT_VERSION_IMAGES}" "true")}
  # Bundle image is in different locations in quay.io and registry.redhat.io
  export SERVERLESS_BUNDLE_REDHAT_IO=${SERVERLESS_BUNDLE_REDHAT_IO//serverless-bundle/serverless-operator-bundle}

  if [[ -z "${INDEX_IMAGE:-}" ]]; then
    # Use the current OCP version if the cluster is running otherwise use the latest.
    if oc get clusterversion &>/dev/null; then
      ocp_version=$(oc get clusterversion version -o jsonpath='{.status.desired.version}')
      ocp_version=$(versions.major_minor "$ocp_version")
    else
      ocp_version=$(metadata.get 'requirements.ocpVersion.min')
    fi
    ocp_version=${ocp_version/./} # 4.17 -> 417

    # Using `export` on the assignment keeps the original semantics of ignoring lookup failures.
    # shellcheck disable=SC2155
    export INDEX_IMAGE=$(latest_konflux_image_sha "${registry_quay}-fbc-${ocp_version}/serverless-index-${quay_registry_app_version}-fbc-${ocp_version}:${CURRENT_VERSION_IMAGES}" "latest" "true")
  fi
  export INDEX_IMAGE
}

# Bundle image is specific as we need to pull older versions for including in the catalog.
function get_bundle_for_version() {
  local version app_version
  version=${1:?"Provide version for Bundle image"}

  app_version=${version/./} # 1.34.0 -> 134.0
  app_version=${app_version%.*} # 134.0 -> 134

  image=$(image_with_sha "${registry_prefix_quay}${app_version}/serverless-bundle:latest")
  image_version=$(bundle_image_version "${registry_prefix_quay}${app_version}/serverless-bundle:latest")

  # When collecting images for images.prefetch, only prefetch the primary location. The fallbacks
  # below are rarely needed and are then looked up on demand.
  if [[ -n "${IMAGE_INSPECT_COLLECT_FILE:-}" ]]; then
    echo "$image"
    return
  fi

  # As a backup, try also CI registry.
  # For .micro releases, it's possible we only have the _previous_ version in Konflux, so also check the version of the bundle
  local ci_bundle="registry.ci.openshift.org/knative/serverless-bundle"
  if [[ "${image}" == "" ]]; then
    image=$(image_with_sha "${ci_bundle}:release-${version}" || echo "")
    image_version=$(bundle_image_version "${ci_bundle}:release-${version}")
  fi
  if [[ "${image}" == "" ]]; then
    image=$(image_with_sha "${ci_bundle}:knative-main")
    image_version=$(bundle_image_version "${ci_bundle}:knative-main")
  fi

  if [[ "${image}" == "" ]]; then
    echo "[ERROR] No image found for $version" > /dev/stderr
    exit 1
  fi

  if [[ "${image_version}" != "${version}" ]]; then
    echo "[WARNING] Image $image $image_version does not match requested version $version" > /dev/stderr
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

  # Test images
  local serving_test="${serving}-test"

  export KNATIVE_SERVING_TEST_HTTPPROXY=${KNATIVE_SERVING_TEST_HTTPPROXY:-$(latest_konflux_image_sha "${serving_test}-httpproxy:${tag}")}
  export KNATIVE_SERVING_TEST_AUTOSCALE=${KNATIVE_SERVING_TEST_AUTOSCALE:-$(latest_konflux_image_sha "${serving_test}-autoscale:${tag}")}
  export KNATIVE_SERVING_TEST_HELLOWORLD=${KNATIVE_SERVING_TEST_HELLOWORLD:-$(latest_konflux_image_sha "${serving_test}-helloworld:${tag}")}
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
  export KNATIVE_EVENTING_AUTH_PROXY=${KNATIVE_EVENTING_AUTH_PROXY:-$(latest_registry_redhat_io_image_sha "${eventing}-auth-proxy:${tag}")}
  export KNATIVE_EVENTING_JOBSINK=${KNATIVE_EVENTING_JOBSINK:-$(latest_registry_redhat_io_image_sha "${eventing}-jobsink:${tag}")}
  export KNATIVE_EVENTING_MIGRATE=${KNATIVE_EVENTING_MIGRATE:-$(latest_registry_redhat_io_image_sha "${eventing}-migrate:${tag}")}
  export KNATIVE_EVENTING_REQUEST_REPLY=${KNATIVE_EVENTING_REQUEST_REPLY:-$(latest_registry_redhat_io_image_sha "${eventing}-requestreply:${tag}")}

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

function knative_eventing_integrations_images_release() {
  knative_eventing_integrations_images "${USE_IMAGE_RELEASE_TAG}"
}

function default_knative_eventing_integrations_images() {
  knative_eventing_integrations_images "$(metadata.get dependencies.eventing_integrations)"
}

function knative_eventing_integrations_images() {
  local eventing_kafka_broker tag app_version
  tag=${1:?"Provide tag for Eventing Integrations images"}

  app_version=$(get_app_version_from_tag "${tag}")
  eventing_integrations="${registry_prefix_quay}${app_version}/kn-eventing-integrations"

  # EventTransform
  export KNATIVE_EVENTING_INTEGRATIONS_TRANSFORM_JSONATA=${KNATIVE_EVENTING_INTEGRATIONS_TRANSFORM_JSONATA:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-transform-jsonata:${tag}")}
  # IntegrationSource
  export KNATIVE_EVENTING_INTEGRATIONS_TIMER_SOURCE=${KNATIVE_EVENTING_INTEGRATIONS_TIMER_SOURCE:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-timer-source:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_S3_SOURCE=${KNATIVE_EVENTING_INTEGRATIONS_AWS_S3_SOURCE:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-s3-source:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_SQS_SOURCE=${KNATIVE_EVENTING_INTEGRATIONS_AWS_SQS_SOURCE:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-sqs-source:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_DDB_STREAMS_SOURCE=${KNATIVE_EVENTING_INTEGRATIONS_AWS_DDB_STREAMS_SOURCE:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-ddb-streams-source:${tag}")}
  # IntegrationSink
  export KNATIVE_EVENTING_INTEGRATIONS_LOG_SINK=${KNATIVE_EVENTING_INTEGRATIONS_LOG_SINK:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-log-sink:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_S3_SINK=${KNATIVE_EVENTING_INTEGRATIONS_AWS_S3_SINK:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-s3-sink:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_SQS_SINK=${KNATIVE_EVENTING_INTEGRATIONS_AWS_SQS_SINK:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-sqs-sink:${tag}")}
  export KNATIVE_EVENTING_INTEGRATIONS_AWS_SNS_SINK=${KNATIVE_EVENTING_INTEGRATIONS_AWS_SNS_SINK:-$(latest_registry_redhat_io_image_sha "${eventing_integrations}-aws-sns-sink:${tag}")}
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

function latest_registry_redhat_io_image_sha() {
  input=${1:?"Provide image"}
  return_input_on_empty=${2:-"false"}

  image_without_tag=${input%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  image=$(image_with_sha "${image_without_tag}:latest" "${return_input_on_empty}")

  if [ "${image}" = "" ]; then
    exit 1
  fi

  if [ "${FORCE_USE_QUAY_IMAGES}" = "true" ]; then
    echo "${image}"
    return
  fi

  rh_registry_image="$(get_rh_registry_image_ref "$image")"
  echo "$rh_registry_image"
}

function latest_konflux_image_sha() {
  input=${1:?"Provide image"}
  tag=${2:-"latest"}
  return_input_on_empty=${3:-"false"}

  image_without_tag=${input%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  image=$(image_with_sha "${image_without_tag}:${tag}" "${return_input_on_empty}")

  if [ "${image}" = "" ]; then
    exit 1
  fi

  echo "${image}"
}

# Prints the file caching the `skopeo inspect` output of the given image (see images.init_inspect_cache).
function image_inspect_cache_file {
  local image="${1:?"Provide image"}"
  # Map characters that are invalid in file names to ones that are invalid in image references, so
  # the mapping is collision-free.
  image="${image//\//+}"
  image="${image//:/=}"
  image="${image//@/,}"
  echo "${IMAGE_INSPECT_CACHE_DIR}/${image}__${TARGET_OS:-linux}_${TARGET_ARCH:-amd64}.json"
}

# Prints the `skopeo inspect` JSON of the given image. Results are cached when
# IMAGE_INSPECT_CACHE_DIR is set. When IMAGE_INSPECT_COLLECT_FILE is set, the image reference is only
# recorded there and placeholder JSON is printed (see images.prefetch).
function image_inspect {
  local image target_os target_arch cache_file output
  image=${1:?"Provide image"}

  # Use TARGET_OS and TARGET_ARCH if set, otherwise default to linux/amd64
  target_os=${TARGET_OS:-linux}
  target_arch=${TARGET_ARCH:-amd64}

  if [[ -n "${IMAGE_INSPECT_CACHE_DIR:-}" ]]; then
    cache_file="$(image_inspect_cache_file "${image}")"
    if [[ -s "${cache_file}" ]]; then
      cat "${cache_file}"
      return
    fi
  fi

  if [[ -n "${IMAGE_INSPECT_COLLECT_FILE:-}" ]]; then
    echo "${image}" >> "${IMAGE_INSPECT_COLLECT_FILE}"
    echo '{"Digest": "sha256:collect", "Labels": {"version": "collect"}}'
    return
  fi

  # shellcheck disable=SC2086
  output=$(skopeo inspect --retry-times=10 --override-os "${target_os}" --override-arch "${target_arch}" --no-tags=true ${SKOPEO_EXTRA_FLAGS} "docker://${image}") || return 1

  if [[ -n "${cache_file:-}" ]]; then
    # Write atomically, as parallel prefetching might write the same file.
    echo "${output}" > "${cache_file}.$$"
    mv -f "${cache_file}.$$" "${cache_file}"
  fi
  echo "${output}"
}

function image_with_sha {
  image=${1:?"Provide image"}
  return_input_on_empty=${2:-"false"}

  digest=$(image_inspect "${image}" | jq -r '.Digest' || echo "")
  if [ "${digest}" = "" ]; then
    if [ "${return_input_on_empty}" = "true" ]; then
      echo "${image}"
    fi
    echo ""
    return
  fi

  image_without_tag=${image%:*} # Remove tag, if any
  image_without_tag=${image_without_tag%@*} # Remove sha, if any

  echo "${image_without_tag}@${digest}"
}

function bundle_image_version {
  image=${1:?"Provide image"}

  version=$(image_inspect "${image}" | jq -r '.Labels.version')
  echo "${version}"
}

function get_app_version_from_tag() {
  local tag app_version
  tag=${1:?"Provide tag for images"}

  if [[ "$tag" == "knative-nightly" ]]; then
    app_version=$quay_registry_app_version
  else
    app_version=$(sobranch --upstream-version "${tag/knative-v/}") # -> release-1.34
    app_version=${app_version/release-/}                   # -> 1.34
    app_version=${app_version/./}                          # -> 134
  fi

  echo "${app_version}"
}

# returns the quay image for a given rh registry image ref
function get_quay_image_ref() {
  local rh_registry_image_ref
  rh_registry_image_ref="${1}"

  if  [[ $rh_registry_image_ref =~ $registry_redhat_io ]]; then
    image=${rh_registry_image_ref##*/} # Get image name after last slash
    # we need to be able to handle cases, when the image has a sha or has a tag
    image_suffix=""
    if [[ "$image" =~ @sha256: ]]; then
      image_suffix="@${image##*@}"
    else
      image_suffix=":${image##*:}"
    fi
    image_name=${image%@*} # Remove sha
    image_name=${image_name%:*} # Remove tag

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

    echo "${registry_quay}/${component}${image_suffix}"
  else
    echo "Image must be from ${registry_redhat_io}, got ${rh_registry_image_ref}"
    return 1
  fi
}

# returns the RH registry image for a given quay image ref
function get_rh_registry_image_ref() {
  local quay_registry_image_ref
  quay_registry_image_ref="${1}"

  if  [[ $quay_registry_image_ref =~ $registry_prefix_quay ]]; then
    image=${quay_registry_image_ref##*/} # Get image name after last slash
    # we need to be able to handle cases, when the image has a sha or has a tag
    image_suffix=""
    if [[ "$image" =~ @sha256: ]]; then
      image_suffix="@${image##*@}"
    else
      image_suffix=":${image##*:}"
    fi
    image_name=${image%@*} # Remove sha
    image_name=${image_name%:*} # Remove tag

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

    echo "${registry_redhat_io}/${image_name}${image_suffix}"
  else
    echo "Image must be from ${registry_quay}, got ${quay_registry_image_ref}"
    return 1
  fi
}
