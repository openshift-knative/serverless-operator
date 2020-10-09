#!/usr/bin/env bash

readonly EVENTING_READY_FILE="/tmp/eventing-prober-ready"
readonly EVENTING_PROBER_FILE="/tmp/eventing-prober-signal"

function upstream_knative_eventing_e2e {
  logger.info 'Running eventing tests'

  local failed=0

  export TEST_IMAGE_TEMPLATE="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  # shellcheck disable=SC1090
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  # run_e2e_tests defined in knative-eventing
  run_e2e_tests || failed=$?

  print_test_result ${failed}

  return $failed
}

function actual_eventing_version {
  oc get knativeeventing.operator.knative.dev \
    knative-eventing -n "${EVENTING_NAMESPACE}" -o=jsonpath="{.status.version}"
}

function run_eventing_preupgrade_test {
  logger.info 'Running Eventing pre upgrade tests'

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  local image_template
  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  go_test_e2e -tags=preupgrade \
    -timeout=10m ./test/upgrade \
    --imagetemplate="${image_template}"

  logger.success 'Eventing pre upgrade tests passed'
}

function start_eventing_prober {
  local eventing_prober_pid result_file image_template
  result_file="${1:?Pass a result file as arg[1]}"

  logger.info 'Starting Eventing prober'

  rm -fv "${EVENTING_PROBER_FILE}" "${EVENTING_READY_FILE}"
  cd "${KNATIVE_EVENTING_HOME}" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  go_test_e2e -tags=probe \
    -timeout=30m \
    ./test/upgrade \
    --pipefile="${EVENTING_PROBER_FILE}" \
    --readyfile="${EVENTING_READY_FILE}" \
    --imagetemplate="${image_template}" &
  eventing_prober_pid=$!

  logger.debug "Eventing prober PID is ${eventing_prober_pid}"

  echo ${eventing_prober_pid} > "${result_file}"
}

function wait_for_eventing_prober_ready {
  wait_for_file "${EVENTING_READY_FILE}"

  logger.success 'Eventing prober is ready'
}

function end_eventing_prober {
  local prober_pid
  prober_pid="${1:?Pass a prober pid as arg[1]}"

  end_prober_test 'Eventing' "${prober_pid}" "${EVENTING_PROBER_FILE}"
}

function run_eventing_postupgrade_test {
  logger.info 'Running Eventing post upgrade tests'
  local image_template

  cd "${KNATIVE_EVENTING_HOME}" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  go_test_e2e -tags=postupgrade \
    -timeout=10m ./test/upgrade \
    --imagetemplate="${image_template}"

  logger.success 'Eventing post upgrade tests passed'
}


