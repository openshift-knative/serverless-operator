#!/usr/bin/env bash

resources_dir="$(dirname "${BASH_SOURCE[0]}")/mesh_resources"
  
function install_mesh {
  ensure_catalog_pods_running
  deploy_servicemesh_operators
  deploy_servicemeshcontrolplane
  deploy_gateways
}

function uninstall_mesh {
  undeploy_gateways
  undeploy_servicemeshcontrolplane
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  if [[ ${SKIP_OPERATOR_SUBSCRIPTION:-} != "true" ]]; then
    logger.info "Installing service mesh operators in namespace openshift-operators"
    oc apply -f "${resources_dir}"/subscription.yaml || return $?
  fi

  logger.info "Waiting until service mesh operators are available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators istio-operator --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Available deployment istio-operator --timeout=300s -n openshift-operators || return $?
}

function undeploy_servicemesh_operators {
  logger.info "Deleting service mesh subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator kiali-ossm jaeger-product --ignore-not-found
  logger.info 'Deleting ClusterServiceVersion'
  for csv in $(set +o pipefail && oc get csv -n openshift-operators --no-headers 2>/dev/null \
      | grep 'servicemeshoperator\|jaeger\|kiali' | cut -f1 -d' '); do
    oc delete csv -n openshift-operators "${csv}"
  done

  logger.info 'Ensure no operators present'
  timeout 600 "[[ \$(oc get deployments -n openshift-operators -oname | grep -c 'servicemeshoperator\|jaeger\|kiali') != 0 ]]"

  logger.info "Deleting service mesh istio nodes"
  oc delete --ignore-not-found=true daemonset.apps/istio-node -n openshift-operators
  oc delete --ignore-not-found=true service/maistra-admission-controller -n openshift-operators

  logger.info "Deleting service mesh webhooks and rbac resources"
  oc delete --ignore-not-found=true validatingwebhookconfiguration openshift-operators.servicemesh-resources.maistra.io
  oc delete --ignore-not-found=true mutatingwebhookconfigurations openshift-operators.servicemesh-resources.maistra.io
  oc delete --ignore-not-found=true clusterrole istio-admin istio-cni istio-edit istio-view
  oc delete --ignore-not-found=true clusterrolebinding istio-cn

  logger.info 'Ensure not CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'maistra.io') -eq 0 ]]; then
    oc get crd -oname | grep 'maistra.io' | xargs oc delete --timeout=60s
  fi
  if [[ ! $(oc get crd -oname | grep -c 'istio') -eq 0 ]]; then
    oc get crd -oname | grep 'istio' | xargs oc delete --timeout=60s
  fi
  logger.success "Service mesh has been uninstalled"
}

function deploy_servicemeshcontrolplane {
  logger.info "Installing ServiceMeshControlPlane in namespace istio-system"

  oc get ns istio-system || oc create namespace istio-system

  # Make sure servicemeshcontrolplanes.maistra.io is available.
  timeout 120 "[[ \$(oc get crd servicemeshcontrolplanes.maistra.io --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Established crd servicemeshcontrolplanes.maistra.io

  # creating smcp often fails due to webhook error
  timeout 120 "[[ \$(oc apply -f ${resources_dir}/smcp.yaml | oc get smcp -n istio-system basic --no-headers | wc -l) != 1 ]]" || return 1

  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platformStatus.aws.resourceTags[?(@.key=="red-hat-clustertype")].value}') = rosa ]]; then
    logger.info "ThirdParty tokens required when using ROSA cluster"
    enable_smcp_third_party_token
  fi

  oc wait --timeout=180s --for=condition=Ready smcp -n istio-system basic || oc get smcp -n istio-system basic -o yaml
}

function undeploy_servicemeshcontrolplane {
  logger.info "Deleting ServiceMeshControlPlane"
  oc delete smcp -n istio-system basic --ignore-not-found || return $?
}

function deploy_gateways {
  # Generate wildcard certs with cluster's subdomain.

  local out_dir
  out_dir="$(mktemp -d /tmp/certs-XXX)"

  openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:2048 \
    -subj "/O=Example Inc./CN=Example" \
    -keyout "${out_dir}"/root.key \
    -out "${out_dir}"/root.crt

  subdomain=$(oc get ingresses.config.openshift.io cluster -o jsonpath="{.spec.domain}")
  openssl req -nodes -newkey rsa:2048 \
      -subj "/O=Example Inc./CN=Example" \
      -reqexts san \
      -config <(printf "[req]\ndistinguished_name=req\n[san]\nsubjectAltName=DNS:*.%s" "$subdomain") \
      -keyout "${out_dir}"/wildcard.key \
      -out "${out_dir}"/wildcard.csr

  openssl x509 -req -days 365 -set_serial 0 \
      -extfile <(printf "subjectAltName=DNS:*.%s" "$subdomain") \
      -CA "${out_dir}"/root.crt \
      -CAkey "${out_dir}"/root.key \
      -in "${out_dir}"/wildcard.csr \
      -out "${out_dir}"/wildcard.crt

  oc create -n istio-system secret tls wildcard-certs \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f - 

  # ca-key-pair secret in cert-manager namespace needs for upstream e2e test with https option.
  oc get ns cert-manager || oc create namespace cert-manager
  oc create -n cert-manager secret tls ca-key-pair \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f -

  oc apply -f "${resources_dir}"/namespace.yaml || return $?
  oc apply -f "${resources_dir}"/smmr.yaml || return $?
  oc apply -f "${resources_dir}"/gateway.yaml || return $?
  oc apply -f "${resources_dir}"/authorization-policies/setup || return $?
  oc apply -f "${resources_dir}"/authorization-policies/helm || return $?
  oc apply -f "${resources_dir}"/destination-rules.yaml || return $?

  oc apply -n "${EVENTING_NAMESPACE}" -f "${resources_dir}"/kafka-service-entry.yaml || return $?
  for ns in serverless-tests eventing-e2e0 eventing-e2e1 eventing-e2e2 eventing-e2e3 eventing-e2e4; do
    oc apply -n "$ns" -f "${resources_dir}"/kafka-service-entry.yaml || return $?
  done
  oc apply -n "serverless-tests" -f "${resources_dir}"/network-policy-monitoring.yaml || return $?
}

function undeploy_gateways {
  oc delete -n serverless-tests -f "${resources_dir}"/network-policy-monitoring.yaml --ignore-not-found || return $?
  for ns in serverless-tests eventing-e2e0 eventing-e2e1 eventing-e2e2 eventing-e2e3 eventing-e2e4; do
    oc delete -n "$ns" -f "${resources_dir}"/kafka-service-entry.yaml --ignore-not-found || return $?
  done
  oc delete -f "${resources_dir}"/destination-rules.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/authorization-policies/helm --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/authorization-policies/setup --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/gateway.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/smmr.yaml --ignore-not-found || return $?
  oc delete -n cert-manager secret ca-key-pair  --ignore-not-found || return $?
  oc delete -n istio-system secret wildcard-certs --ignore-not-found || return $?
}

function enable_smcp_third_party_token {
  smcp_patch="$(mktemp -t smcp-XXXXX.yaml)"

  cat <<EOF > "${smcp_patch}"
spec:
  security:
    identity:
      type: ThirdParty
EOF

  oc patch smcp -n istio-system basic --type='merge' --patch-file "${smcp_patch}"
}
