#!/usr/bin/env bash

export BUILD_NUMBER=${BUILD_NUMBER:-$(head -c 128 </dev/urandom | basenc --base64url | fold -w 8 | head -n 1)}

if [[ -n "${ARTIFACT_DIR:-}" ]]; then
  ARTIFACTS="${ARTIFACT_DIR}/build-${BUILD_NUMBER}"
  export ARTIFACTS
  mkdir -p "${ARTIFACTS}"
fi

# shellcheck disable=SC1091,SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../../vendor/knative.dev/hack/e2e-tests.sh"
source "$(dirname "${BASH_SOURCE[0]}")/images.bash"

export STRIMZI_VERSION=0.45.1

# Adjust these when upgrading the knative versions.
export KNATIVE_SERVING_VERSION="${KNATIVE_SERVING_VERSION:-$(metadata.get dependencies.serving)}"
export KNATIVE_SERVING_VERSION_PREVIOUS="${KNATIVE_SERVING_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.serving)}"
export KNATIVE_EVENTING_VERSION="${KNATIVE_EVENTING_VERSION:-$(metadata.get dependencies.eventing)}"
export KNATIVE_EVENTING_VERSION_PREVIOUS="${KNATIVE_EVENTING_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.eventing)}"
export KNATIVE_EVENTING_KAFKA_BROKER_VERSION="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION:-$(metadata.get dependencies.eventing_kafka_broker)}"
export KNATIVE_EVENTING_ISTIO_VERSION="${KNATIVE_EVENTING_ISTIO_VERSION:-$(metadata.get dependencies.eventing_istio)}"
export KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS="${KNATIVE_EVENTING_KAFKA_BROKER_VERSION_PREVIOUS:-$(metadata.get dependencies.previous.eventing_kafka_broker)}"

CURRENT_VERSION="$(metadata.get project.version)"
PREVIOUS_VERSION="$(metadata.get olm.replaces)"
CURRENT_CSV="$(metadata.get project.name).v$CURRENT_VERSION"
PREVIOUS_CSV="$(metadata.get project.name).v$PREVIOUS_VERSION"
export CURRENT_VERSION PREVIOUS_VERSION CURRENT_CSV PREVIOUS_CSV

# Directories below are filled with source code by ci-operator
export TEST_SOURCE_BASE_PATH=${TEST_SOURCE_BASE_PATH:-"/go/src/knative.dev"}
export KNATIVE_SERVING_HOME="${TEST_SOURCE_BASE_PATH}/serving"
export KNATIVE_EVENTING_HOME="${TEST_SOURCE_BASE_PATH}/eventing"
export KNATIVE_EVENTING_KAFKA_BROKER_HOME="${TEST_SOURCE_BASE_PATH}/eventing-kafka-broker"
export KNATIVE_EVENTING_ISTIO_HOME="${TEST_SOURCE_BASE_PATH}/eventing-istio"
export BROKER_CLASS=${BROKER_CLASS:-"Kafka"}

export DOCKER_REPO_OVERRIDE="${DOCKER_REPO_OVERRIDE:-}"
export INTERACTIVE="${INTERACTIVE:-$(
  test -z "${GDMSESSION}"
  echo $?
)}"
export KUBECONFIG="${KUBECONFIG:-$(realpath ~/.kube/config)}"
export OPENSHIFT_CI="${OPENSHIFT_CI:-}"
export OPERATOR="${OPERATOR:-serverless-operator}"
export SCALE_UP="${SCALE_UP:--1}"

export OLM_NAMESPACE="${OLM_NAMESPACE:-openshift-marketplace}"
export ON_CLUSTER_BUILDS_NAMESPACE="${ON_CLUSTER_BUILDS_NAMESPACE:-openshift-serverless-builds}"
export OPERATORS_NAMESPACE="${OPERATORS_NAMESPACE:-openshift-serverless}"
export SERVING_NAMESPACE="${SERVING_NAMESPACE:-knative-serving}"
export INGRESS_NAMESPACE="${INGRESS_NAMESPACE:-knative-serving-ingress}"
export EVENTING_NAMESPACE="${EVENTING_NAMESPACE:-knative-eventing}"
# eventing e2e and conformance tests use a container for tracing tests that has hardcoded to `knative-tracing` in it
export TRACING_NAMESPACE="${TRACING_NAMESPACE:-knative-tracing}"
export TRACING_BACKEND="${TRACING_BACKEND:-tempo}"

declare -a SYSTEM_NAMESPACES
SYSTEM_NAMESPACES=("${TRACING_NAMESPACE}" "${OPERATORS_NAMESPACE}" "${ON_CLUSTER_BUILDS_NAMESPACE}")
export SYSTEM_NAMESPACES
export UPGRADE_SERVERLESS="${UPGRADE_SERVERLESS:-"true"}"
export UPGRADE_CLUSTER="${UPGRADE_CLUSTER:-"false"}"
# Change this when forcing the upgrade to an image that is not yet available via upgrade channel
export UPGRADE_OCP_IMAGE="${UPGRADE_OCP_IMAGE:-}"

export INSTALL_PREVIOUS_VERSION="${INSTALL_PREVIOUS_VERSION:-"false"}"
export INSTALL_OLDEST_COMPATIBLE="${INSTALL_OLDEST_COMPATIBLE:-"false"}"

OLM_CHANNEL="${OLM_CHANNEL:-$(metadata.get olm.channels.default)}"
export OLM_CHANNEL
# Change this when upgrades need switching to a different channel
export OLM_UPGRADE_CHANNEL="${OLM_UPGRADE_CHANNEL:-"$OLM_CHANNEL"}"
export OLM_SOURCE="${OLM_SOURCE:-"$OPERATOR"}"

# OLM version control (v0 or v1)
export OLM_VERSION="${OLM_VERSION:-v0}"

# OLMv1-specific configuration
export OLMV1_CATALOG_NAME="${OLMV1_CATALOG_NAME:-serverless-operator}"
export OLMV1_CATALOG_PRIORITY="${OLMV1_CATALOG_PRIORITY:-10}"
export OLMV1_INSTALLER_SA="${OLMV1_INSTALLER_SA:-serverless-operator-installer}"
export OLMV1_CLUSTEREXTENSION_NAME="${OLMV1_CLUSTEREXTENSION_NAME:-serverless-operator}"
export OLMV1_UPGRADE_CONSTRAINT_POLICY="${OLMV1_UPGRADE_CONSTRAINT_POLICY:-CatalogProvided}"

# OLMv1 system namespaces
export CATALOGD_NAMESPACE="${CATALOGD_NAMESPACE:-openshift-catalogd}"
export OPERATOR_CONTROLLER_NAMESPACE="${OPERATOR_CONTROLLER_NAMESPACE:-openshift-operator-controller}"

export TEST_KNATIVE_UPGRADE="${TEST_KNATIVE_UPGRADE:-true}"
export TEST_KNATIVE_E2E="${TEST_KNATIVE_E2E:-true}"
export TEST_KNATIVE_SERVING="${TEST_KNATIVE_SERVING:-false}"
export TEST_KNATIVE_EVENTING="${TEST_KNATIVE_EVENTING:-false}"
export TEST_KNATIVE_KAFKA="${TEST_KNATIVE_KAFKA:-false}"
export TEST_KNATIVE_KAFKA_BROKER="${TEST_KNATIVE_KAFKA_BROKER:-false}"

# Makefile triggers for modular install
export INSTALL_CERTMANAGER="${INSTALL_CERTMANAGER:-true}"
export INSTALL_SERVING="${INSTALL_SERVING:-true}"
export INSTALL_EVENTING="${INSTALL_EVENTING:-true}"
export INSTALL_KAFKA="${INSTALL_KAFKA:-false}"
export MESH="${MESH:-false}"
export ENABLE_TRACING="${ENABLE_TRACING:-false}"
export ENABLE_KEDA="${ENABLE_KEDA:-false}"
# Define sample-rate for tracing.
export SAMPLE_RATE="${SAMPLE_RATE:-"1.0"}"
export ZIPKIN_DEDICATED_NODE="${ZIPKIN_DEDICATED_NODE:-false}"
export QUAY_REGISTRY=quay.io/openshift-knative

default_knative_serving_images
default_knative_eventing_images

DEFAULT_IMAGE_TEMPLATE=$(
  cat <<-EOF
{{- with .Name }}
{{- if eq . "httpproxy" }}${KNATIVE_SERVING_TEST_HTTPPROXY}
{{- else if eq . "autoscale" }}${KNATIVE_SERVING_TEST_AUTOSCALE}
{{- else if eq . "helloworld-go" }}${KNATIVE_SERVING_TEST_HELLOWORLD}
{{- else if eq . "recordevents" }}${KNATIVE_EVENTING_TEST_RECORDEVENTS}
{{- else if eq . "wathola-forwarder" }}${KNATIVE_EVENTING_TEST_WATHOLA_FORWARDER}
{{- else if eq . "kafka" }}quay.io/strimzi/kafka:latest-kafka-3.4.0
{{- else }}${KNATIVE_SERVING_IMAGE_PREFIX}-test-{{.}}:latest{{end -}}
{{end -}}
EOF
)
export IMAGE_TEMPLATE="${IMAGE_TEMPLATE:-"$DEFAULT_IMAGE_TEMPLATE"}"
export INDEX_IMAGE="${INDEX_IMAGE:-}"
# Might be used to disable tests which need additional users.
# Managed environments such as Hypershift might now allow creating new users.
export USER_MANAGEMENT_ALLOWED="${USER_MANAGEMENT_ALLOWED:-true}"
export DELETE_CRD_ON_TEARDOWN="${DELETE_CRD_ON_TEARDOWN:-true}"
export USE_RELEASED_HELM_CHART="${USE_RELEASED_HELM_CHART:-false}"
export HELM_CHART_TGZ="${HELM_CHART_TGZ:-}"
export HA="${HA:-true}"
export USE_RELEASE_NEXT="${USE_RELEASE_NEXT:-false}"
export USE_IMAGE_RELEASE_TAG="${USE_IMAGE_RELEASE_TAG:-}"
export USE_ARTIFACTS_RELEASE_BRANCH="${USE_ARTIFACTS_RELEASE_BRANCH:-}"

if [ "${USE_IMAGE_RELEASE_TAG}" != "" ] && [ "${USE_ARTIFACTS_RELEASE_BRANCH}" = "" ]; then
   if [ "${USE_IMAGE_RELEASE_TAG}" = "knative-nightly" ]; then
     export USE_ARTIFACTS_RELEASE_BRANCH="release-next"
   else
     export USE_ARTIFACTS_RELEASE_BRANCH=${USE_IMAGE_RELEASE_TAG/knative-/release-}
   fi
fi

if [ "${USE_RELEASE_NEXT}" = "true" ]; then
  export USE_IMAGE_RELEASE_TAG="knative-nightly"
  export USE_ARTIFACTS_RELEASE_BRANCH="release-next"
fi

echo "Branch and Tag: ${USE_ARTIFACTS_RELEASE_BRANCH} - ${USE_IMAGE_RELEASE_TAG}"

if [ "${USE_IMAGE_RELEASE_TAG}" != "" ] && [ "${USE_ARTIFACTS_RELEASE_BRANCH}" != "" ]; then
  root_dir="$(dirname "$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")")"
  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing' "${USE_IMAGE_RELEASE_TAG}"
  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing_artifacts_branch' "${USE_ARTIFACTS_RELEASE_BRANCH}"

  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing_kafka_broker' "${USE_IMAGE_RELEASE_TAG}"
  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing_kafka_broker_artifacts_branch' "${USE_ARTIFACTS_RELEASE_BRANCH}"

  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing_istio' "${USE_IMAGE_RELEASE_TAG}"
  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.eventing_istio_artifacts_branch' "${USE_ARTIFACTS_RELEASE_BRANCH}"

  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.serving' "${USE_IMAGE_RELEASE_TAG}"
  yq w --inplace "${root_dir}/olm-catalog/serverless-operator/project.yaml" 'dependencies.serving_artifacts_branch' "${USE_ARTIFACTS_RELEASE_BRANCH}"
fi

# Waits until the given object exists.
# Parameters: $1 - the kind of the object.
#             $2 - object's name.
#             $3 - namespace (optional).
# shellcheck disable=SC2034,SC2086,SC2086
function wait_until_object_exists() {
  local KUBECTL_ARGS="get $1 $2"
  local DESCRIPTION="$1 $2"

  if [[ -n $3 ]]; then
    KUBECTL_ARGS="get -n $3 $1 $2"
    DESCRIPTION="$1 $3/$2"
  fi
  echo -n "Waiting until ${DESCRIPTION} exists"
  for i in {1..150}; do  # timeout after 5 minutes
    if kubectl ${KUBECTL_ARGS} > /dev/null 2>&1; then
      echo -e "\n${DESCRIPTION} exists"
      return 0
    fi
    echo -n "."
    sleep 2
  done
  echo -e "\n\nERROR: timeout waiting for ${DESCRIPTION} to exist"
  kubectl ${KUBECTL_ARGS}
  return 1
}

