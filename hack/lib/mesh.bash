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
  timeout 600 "[[ \$(oc get deploy -n openshift-operators sail-operator --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Available deployment sail-operator --timeout=300s -n openshift-operators || return $?
}

function undeploy_servicemesh_operators {
  logger.info "Deleting service mesh subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators sailoperator --ignore-not-found
  logger.info 'Deleting ClusterServiceVersion'
  for csv in $(set +o pipefail && oc get csv -n openshift-operators --no-headers 2>/dev/null \
      | grep 'sailoperator' | cut -f1 -d' '); do
    oc delete csv -n openshift-operators "${csv}"
  done

  logger.info 'Ensure no operators present'
  timeout 600 "[[ \$(oc get deployments -n openshift-operators -oname | grep -c 'sail-operator') != 0 ]]"

  logger.info "Deleting service mesh webhooks and rbac resources"
  oc delete --ignore-not-found=true clusterrole istio-admin istio-edit istio-view

  logger.info 'Ensure not CRDs left'
  if [[ ! $(oc get crd -oname | grep -c 'istio') -eq 0 ]]; then
    oc get crd -oname | grep 'istio' | xargs oc delete --timeout=60s
  fi
  logger.success "Service mesh has been uninstalled"
}

function deploy_servicemeshcontrolplane {
  logger.info "Installing istiod in namespace istio-system"

  oc get ns istio-system || oc create namespace istio-system
  oc get ns istio-cni || oc create namespace istio-cni

  # Make sure istios.operator.istio.io is available.
  timeout 120 "[[ \$(oc get crd istios.operator.istio.io --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Established crd istios.operator.istio.io

  timeout 120 "[[ \$(oc apply -f ${resources_dir}/istio.yaml | oc get istios -n istio-system default --no-headers | wc -l) != 1 ]]" || return 1
  timeout 120 "[[ \$(oc apply -f ${resources_dir}/istio-cni.yaml | oc get istiocnis -n default default --no-headers | wc -l) != 1 ]]" || return 1

# TODO: CHECK ME for OSSM3
#  if [[ $(oc get infrastructure cluster -ojsonpath='{.status.platformStatus.aws.resourceTags[?(@.key=="red-hat-clustertype")].value}') = rosa ]]; then
#    logger.info "ThirdParty tokens required when using ROSA cluster"
#    enable_smcp_third_party_token
#  fi

  oc wait --timeout=180s --for=condition=Ready istios -n istio-system default || oc get istios -n istio-system default -o yaml
  oc wait --timeout=180s --for=condition=Ready istiocnis -n default default || oc get istiocnis -n default default -o yaml

  # make sure istiod + cni pods are up before continuing
  oc wait deploy --all --timeout=600s --for=condition=Available -n istio-system
  oc rollout status daemonset -n istio-cni --timeout 600s
}

function undeploy_servicemeshcontrolplane {
  logger.info "Deleting istiod"
  oc delete istios default -n istio-system --ignore-not-found || return $?
  oc delete istiocnis default --ignore-not-found || return $?
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

  oc apply -f "${resources_dir}"/namespace.yaml || return $?

  oc create -n knative-serving-ingress secret tls wildcard-certs \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f - 

  # ca-key-pair secret in cert-manager namespace needs for upstream e2e test with https option.
  oc get ns cert-manager || oc create namespace cert-manager
  oc create -n cert-manager secret tls ca-key-pair \
      --key="${out_dir}"/wildcard.key \
      --cert="${out_dir}"/wildcard.crt --dry-run=client -o yaml | oc apply -f -

  oc apply -f "${resources_dir}"/gateway-deploy.yaml || return $?
  oc apply -f "${resources_dir}"/gateway.yaml || return $?
  oc apply -f "${resources_dir}"/authorization-policies/setup || return $?
  oc apply -f "${resources_dir}"/authorization-policies/helm || return $?
  oc apply -f "${resources_dir}"/destination-rules.yaml || return $?
  oc apply -f "${resources_dir}"/peer-authentication-mesh-mtls.yaml || return $?

  oc apply -n "${EVENTING_NAMESPACE}" -f "${resources_dir}"/kafka-service-entry.yaml || return $?
  for ns in serverless-tests eventing-e2e0 eventing-e2e1 eventing-e2e2 eventing-e2e3 eventing-e2e4; do
    oc apply -n "$ns" -f "${resources_dir}"/kafka-service-entry.yaml || return $?
  done
}

function undeploy_gateways {
  for ns in serverless-tests eventing-e2e0 eventing-e2e1 eventing-e2e2 eventing-e2e3 eventing-e2e4; do
    oc delete -n "$ns" -f "${resources_dir}"/kafka-service-entry.yaml --ignore-not-found || return $?
  done
  oc delete -f "${resources_dir}"/peer-authentication-mesh-mtls.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/destination-rules.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/authorization-policies/helm --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/authorization-policies/setup --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/gateway.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/gateway-deploy.yaml --ignore-not-found || return $?
  oc delete -n cert-manager secret ca-key-pair  --ignore-not-found || return $?
  oc delete -n knative-serving-ingress secret wildcard-certs --ignore-not-found || return $?
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
