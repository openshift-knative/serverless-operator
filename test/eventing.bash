#!/usr/bin/env bash

function knative_eventing_tests {
  (
  local failed=0
  logger.info 'Running eventing tests'

  cd "$KNATIVE_EVENTING_HOME" || return $?

  image_template="registry.svc.ci.openshift.org/openshift/knative-${KNATIVE_EVENTING_VERSION}:knative-eventing-test-{{.Name}}"

  oc patch KnativeEventing knative-eventing \
    --namespace ${EVENTING_NAMESPACE} \
    --type merge \
    --patch '{"spec":{"defaultBrokerClass":"ChannelBasedBroker"}}' || failed=1

  go_test_e2e -timeout=90m -parallel=12 ./test/e2e -brokerclass=ChannelBasedBroker -channels=messaging.knative.dev/v1alpha1:InMemoryChannel,messaging.knative.dev/v1alpha1:Channel,messaging.knative.dev/v1beta1:InMemoryChannel \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=2

  oc patch KnativeEventing knative-eventing \
    --namespace ${EVENTING_NAMESPACE} \
    --type merge \
    --patch '{"spec":{"defaultBrokerClass":"MTChannelBasedBroker"}}' || failed=3

  go_test_e2e -timeout=90m -parallel=12 ./test/e2e -brokerclass=MTChannelBasedBroker -channels=messaging.knative.dev/v1alpha1:InMemoryChannel,messaging.knative.dev/v1alpha1:Channel,messaging.knative.dev/v1beta1:InMemoryChannel \
    --kubeconfig "$KUBECONFIG" \
    --imagetemplate "$image_template" || failed=4


  print_test_result ${failed}

  return $failed
  )
}
