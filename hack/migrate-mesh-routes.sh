#!/usr/bin/env bash

# Migrate OpenShift Routes from SM2 (istio-system) to SM3 (knative-serving-ingress).
#
# SM2 created routes in istio-system targeting the istio-ingressgateway service.
# SM3 uses knative-istio-ingressgateway in knative-serving-ingress instead.
# The knative-openshift-ingress controller has a bug where stale routes are not
# cleaned up when the gateway namespace changes (map keyed by name, not namespace/name).
# This script finds and deletes those stale routes so the controller can reconcile
# correctly in the new namespace.

set -euo pipefail

readonly SM2_NAMESPACE="${SM2_NAMESPACE:-istio-system}"
readonly SM3_NAMESPACE="${SM3_NAMESPACE:-knative-serving-ingress}"
readonly SM3_SERVICE="${SM3_SERVICE:-knative-istio-ingressgateway}"
readonly LABEL_SELECTOR="serving.knative.openshift.io/ingressName"
readonly DRY_RUN="${DRY_RUN:-false}"

info()  { echo "[INFO]  $*"; }
warn()  { echo "[WARN]  $*"; }
error() { echo "[ERROR] $*" >&2; }

# Find all routes in SM2_NAMESPACE created by the knative-openshift-ingress controller.
find_stale_routes() {
  oc get routes -n "${SM2_NAMESPACE}" -l "${LABEL_SELECTOR}" \
    -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.host}{"\t"}{.spec.to.name}{"\n"}{end}' 2>/dev/null
}

# Check if a matching route already exists in SM3_NAMESPACE.
route_exists_in_sm3() {
  local name="$1"
  oc get route -n "${SM3_NAMESPACE}" "${name}" &>/dev/null
}

# Check route admission status.
route_is_admitted() {
  local ns="$1" name="$2"
  local admitted
  admitted=$(oc get route -n "${ns}" "${name}" \
    -o jsonpath='{.status.ingress[0].conditions[?(@.type=="Admitted")].status}' 2>/dev/null)
  [[ "${admitted}" == "True" ]]
}

main() {
  info "=== SM2 → SM3 Route Migration ==="
  info "Source namespace:      ${SM2_NAMESPACE}"
  info "Destination namespace: ${SM3_NAMESPACE}"
  info "Target service:        ${SM3_SERVICE}"
  info "Dry run:               ${DRY_RUN}"
  echo

  # Verify SM3 service exists.
  if ! oc get svc -n "${SM3_NAMESPACE}" "${SM3_SERVICE}" &>/dev/null; then
    error "SM3 ingress gateway service '${SM3_SERVICE}' not found in '${SM3_NAMESPACE}'."
    error "Ensure SM3 gateways are deployed before running this migration."
    exit 1
  fi

  local stale_routes
  stale_routes=$(find_stale_routes)

  if [[ -z "${stale_routes}" ]]; then
    info "No stale routes found in '${SM2_NAMESPACE}'. Nothing to migrate."
    exit 0
  fi

  local total=0 deleted=0 skipped=0 errors=0

  while IFS=$'\t' read -r name host service; do
    [[ -z "${name}" ]] && continue
    total=$((total + 1))

    info "Route: ${name}"
    info "  Host:    ${host}"
    info "  Service: ${service} (in ${SM2_NAMESPACE})"

    # Check if the target service exists in SM2_NAMESPACE (it shouldn't for SM3).
    if oc get svc -n "${SM2_NAMESPACE}" "${service}" &>/dev/null; then
      warn "  Service '${service}' still exists in '${SM2_NAMESPACE}' — skipping (may still be SM2)."
      skipped=$((skipped + 1))
      continue
    fi

    # Check if a replacement route already exists in SM3_NAMESPACE.
    if route_exists_in_sm3 "${name}"; then
      info "  Replacement route already exists in '${SM3_NAMESPACE}'."

      if route_is_admitted "${SM3_NAMESPACE}" "${name}"; then
        info "  SM3 route is already admitted — stale route may have been cleaned up."
      else
        info "  SM3 route has HostAlreadyClaimed — deleting stale route will fix this."
      fi
    else
      info "  No replacement route in '${SM3_NAMESPACE}' yet — controller will create it after cleanup."
    fi

    if [[ "${DRY_RUN}" == "true" ]]; then
      info "  [DRY RUN] Would delete route '${name}' from '${SM2_NAMESPACE}'."
    else
      info "  Deleting stale route '${name}' from '${SM2_NAMESPACE}'..."
      if oc delete route -n "${SM2_NAMESPACE}" "${name}"; then
        deleted=$((deleted + 1))
        info "  Deleted successfully."
      else
        error "  Failed to delete route '${name}'."
        errors=$((errors + 1))
      fi
    fi
    echo
  done <<< "${stale_routes}"

  info "=== Migration Summary ==="
  info "Total stale routes found: ${total}"
  if [[ "${DRY_RUN}" == "true" ]]; then
    info "Would delete: ${total} (dry run)"
  else
    info "Deleted:  ${deleted}"
  fi
  info "Skipped:  ${skipped}"
  info "Errors:   ${errors}"

  if [[ "${DRY_RUN}" != "true" && ${deleted} -gt 0 ]]; then
    echo
    info "Waiting for route reconciliation..."
    sleep 5

    # Verify SM3 routes are now admitted.
    local admitted=0
    while IFS=$'\t' read -r name host service; do
      [[ -z "${name}" ]] && continue
      if route_is_admitted "${SM3_NAMESPACE}" "${name}"; then
        info "  ✓ ${name} (${host}) — admitted in ${SM3_NAMESPACE}"
        admitted=$((admitted + 1))
      elif route_exists_in_sm3 "${name}"; then
        warn "  ⏳ ${name} (${host}) — exists but not yet admitted"
      else
        warn "  ⏳ ${name} (${host}) — not yet created, controller will reconcile"
      fi
    done <<< "${stale_routes}"

    if [[ ${admitted} -eq ${deleted} ]]; then
      info "All migrated routes are admitted."
    else
      info "Some routes may need more time. Re-run or check with:"
      info "  oc get routes -n ${SM3_NAMESPACE} -l ${LABEL_SELECTOR}"
    fi
  fi

  [[ ${errors} -eq 0 ]]
}

main "$@"
