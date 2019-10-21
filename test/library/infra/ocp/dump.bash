#!/usr/bin/env bash

include ui/logger.bash
include logic/facts.bash

function dump_openshift_olm_state {
  logger.info "Dump of subscriptions.operators.coreos.com"
  # This is for status checking.
  oc get subscriptions.operators.coreos.com -o yaml --all-namespaces
  logger.info "Dump of catalog operator log"
  oc logs -n openshift-operator-lifecycle-manager deployment/catalog-operator
}

function dump_openshift_ingress_state {
  logger.info "Dump of routes.route.openshift.io"
  oc get routes.route.openshift.io -o yaml --all-namespaces
  logger.info "Dump of routes.serving.knative.dev"
  oc get routes.serving.knative.dev -o yaml --all-namespaces
  logger.info "Dump of openshift-ingress log"
  oc logs deployment/knative-openshift-ingress -n "$SERVING_NAMESPACE"
}

function dump_state {
  if (( CI )); then
    logger.info 'Skipping dump because running as interactive user'
    return 0
  fi
  logger.info 'Environment'
  env
  
  dump_cluster_state
  dump_openshift_olm_state
  dump_openshift_ingress_state
}
