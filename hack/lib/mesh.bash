#!/usr/bin/env bash

resources_dir="$(dirname "${BASH_SOURCE[0]}")/mesh_resources"
  
function install_mesh {
  deploy_servicemesh_operators
  if [[ ${FULL_MESH:-} == "true" ]]; then
    deploy_servicemeshcontrolplane
    deploy_gateways
  fi
}

function uninstall_mesh {
  if [[ ${FULL_MESH:-} == "true" ]]; then
    undeploy_gateways
    undeploy_servicemeshcontrolplane
  fi
  undeploy_servicemesh_operators
}

function deploy_servicemesh_operators {
  logger.info "Installing service mesh operators in namespace openshift-operators"
  logger.info "Operator source is $OLM_SOURCE"
  sed -i "s|source: .*|source: $OLM_SOURCE|g" "${resources_dir}"/subscription.yaml
  oc apply -f "${resources_dir}"/subscription.yaml || return $?

  logger.info "Waiting until service mesh operators are available"
  timeout 600 "[[ \$(oc get deploy -n openshift-operators istio-operator --no-headers | wc -l) != 1 ]]" || return 1
  oc wait --for=condition=Available deployment istio-operator --timeout=300s -n openshift-operators || return $?
}

function undeploy_servicemesh_operators {
  logger.warn 'Teardown service mesh'
  logger.info "Deleting service mesh CSVs"
  if oc get subscription.operators.coreos.com servicemeshoperator -n openshift-operators >/dev/null 2>&1; then
    CSV=$(oc get subscription.operators.coreos.com servicemeshoperator -n openshift-operators -o=custom-columns=CURRENT_CSV:.status.currentCSV --no-headers=true)
    oc delete --ignore-not-found=true clusterserviceversions.operators.coreos.com $CSV -n openshift-operators
  fi
  if oc get subscription.operators.coreos.com kiali-ossm -n openshift-operators >/dev/null 2>&1; then
    CSV=$(oc get subscription.operators.coreos.com kiali-ossm -n openshift-operators -o=custom-columns=CURRENT_CSV:.status.currentCSV --no-headers=true)
    oc delete --ignore-not-found=true clusterserviceversions.operators.coreos.com $CSV -n openshift-operators
  fi
  if oc get subscription.operators.coreos.com jaeger-product -n openshift-operators >/dev/null 2>&1; then
    CSV=$(oc get subscription.operators.coreos.com jaeger-product -n openshift-operators -o=custom-columns=CURRENT_CSV:.status.currentCSV --no-headers=true)
    oc delete --ignore-not-found=true clusterserviceversions.operators.coreos.com $CSV -n openshift-operators
  fi

  logger.info "Deleting service mesh istio nodes"
  oc delete --ignore-not-found=true daemonset.apps/istio-node -n openshift-operators
  oc delete --ignore-not-found=true service/maistra-admission-controller -n openshift-operators

  logger.info "Deleting service mesh webhooks and rbac resources"
  oc delete --ignore-not-found=true validatingwebhookconfiguration openshift-operators.servicemesh-resources.maistra.io
  oc delete --ignore-not-found=true mutatingwebhookconfigurations openshift-operators.servicemesh-resources.maistra.io
  oc delete --ignore-not-found=true clusterrole istio-admin istio-cni istio-edit istio-view
  oc delete --ignore-not-found=true clusterrolebinding istio-cni

  logger.info "Deleting service mesh subscriptions"
  oc delete subscriptions.operators.coreos.com -n openshift-operators servicemeshoperator kiali-ossm jaeger-product --ignore-not-found

  logger.info "Deleting maistra CRDs"
  if oc get crds -oname | grep -q 'maistra.io'; then
    oc get crds -oname | grep 'maistra.io' | xargs -r oc delete
  fi

  logger.info "Deleting istio CRDs"
  if oc get crds -oname | grep -q 'istio'; then
    oc get crds -oname | grep 'istio' | xargs -r oc delete
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
  oc wait --timeout=180s --for=condition=Ready smcp -n istio-system basic || oc get smcp -n istio-system basic -o yaml
}

function undeploy_servicemeshcontrolplane {
  logger.info "Deleting ServiceMeshControlPlane"
  oc delete smcp -n openshift-operators basic --ignore-not-found || return $?
}

function deploy_gateways {
  oc apply -f "${resources_dir}"/smmr.yaml || return $?

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
  oc apply -f "${resources_dir}"/gateway.yaml || return $?
  oc apply -f "${resources_dir}"/peerauthentication.yaml || return $?
}

function undeploy_gateways {
  oc delete -f "${resources_dir}"/peerauthentication.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/gateway.yaml --ignore-not-found || return $?
  oc delete -f "${resources_dir}"/smmr.yaml --ignore-not-found || return $?
  oc delete -n cert-manager secret ca-key-pair  --ignore-not-found || return $?
  oc delete -n istio-system secret wildcard-certs --ignore-not-found || return $?
}
