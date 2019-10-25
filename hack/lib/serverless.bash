#!/usr/bin/env bash

function ensure_serverless_installed {
  logger.info 'Check if Serverless is installed'
  if oc get knativeserving knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.success 'Serverless is already installed.'
    return 0
  fi
  install_serverless
}

function install_serverless {
  logger.info 'Install the Serverless Operator'
  cat <<EOF | oc apply -f - || return $?
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: ${OPERATOR}
  namespace: ${OPERATORS_NAMESPACE}
spec:
  channel: techpreview
  name: ${OPERATOR}
  source: ${OPERATOR}
  sourceNamespace: ${OPERATORS_NAMESPACE}
EOF
  logger.info "Wait for the ${OPERATOR} pod to appear"
  timeout 900 "[[ \$(oc get pods -n ${OPERATORS_NAMESPACE} | grep -c ${OPERATOR}) -eq 0 ]]" || return 5

  logger.info 'Wait until the Operator pod is up and running'
  wait_until_pods_running "${OPERATORS_NAMESPACE}" || return 6

  logger.info 'Wait until Operator finishes installation'
  timeout 300 "[[ \$(oc get csv -n ${OPERATORS_NAMESPACE} | grep ${OPERATOR} | grep -c Succeeded) -eq 0 ]]" || return 7

  logger.info 'Deploy Knative Serving'
  cat <<EOF | oc apply -f - || return $?
apiVersion: serving.knative.dev/v1alpha1
kind: KnativeServing
metadata:
  name: knative-serving
  namespace: ${SERVING_NAMESPACE}
EOF
  wait_until_pods_running "${SERVING_NAMESPACE}" || return 8

  logger.success 'Serverless has been installed sucessfully.'
}

function teardown_serverless {
  logger.warn '😭  Teardown Serverless...'

  if oc get knativeserving knative-serving -n "${SERVING_NAMESPACE}" >/dev/null 2>&1; then
    logger.info 'Removing KnativeServing CR'
    oc delete knativeserving knative-serving -n "${SERVING_NAMESPACE}" || return $?

    logger.info 'Wait until there are no knative serving pods running'
    timeout 600 "[[ \$(oc get pods -n ${SERVING_NAMESPACE} -o jsonpath='{.items}') != '[]' ]]" || return 9
  fi

  logger.success 'Serverless has been uninstalled.'
}
