#!/usr/bin/env bash

# For SC2164
set -e

readonly EVENTING_READY_FILE="/tmp/eventing-prober-ready"
readonly EVENTING_PROBER_FILE="/tmp/eventing-prober-signal"

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "${KNATIVE_EVENTING_HOME}"

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests
}

function actual_eventing_version {
  oc get knativeeventing.operator.knative.dev \
    knative-eventing -n "${EVENTING_NAMESPACE}" -o=jsonpath="{.status.version}"
}

function prepare_knative_eventing_tests {
  logger.info 'Nothing to prepare for Eventing upgrade tests'
}

function check_eventing_upgraded {
  local latest_version
  latest_version="${1:?Pass a target eventing version as arg[1]}"

  logger.debug 'Check KnativeEventing has the latest version with Ready status'
  timeout 300 "[[ ! ( \$(oc get knativeeventing.operator.knative.dev \
    knative-eventing -n ${EVENTING_NAMESPACE} -o=jsonpath='{.status.version}') \
    == ${latest_version} && \$(oc get knativeeventing.operator.knative.dev \
    knative-eventing -n ${EVENTING_NAMESPACE} \
    -o=jsonpath='{.status.conditions[?(@.type==\"Ready\")].status}') == True ) ]]"
}
