package spoof

import (
	"fmt"
	"net/http"
	"strings"
)

// isUnknownAuthority checks if the error contains "certificate signed by unknown authority".
// This error happens when OpenShift Route starts/changes to use passthrough mode. It takes a little bit time to be synced.
func isUnknownAuthority(err error) bool {
	return err != nil && strings.Contains(err.Error(), "certificate signed by unknown authority")
}

// RetryingRouteInconsistency retries common requests seen when creating a new route
// - 503 to account for Openshift route inconsistency (https://jira.coreos.com/browse/SRVKS-157)
func RouteInconsistencyRetryChecker(resp *Response) (bool, error) {
	if resp.StatusCode == http.StatusServiceUnavailable {
		return true, fmt.Errorf("retrying route inconsistency request: %s", resp)
	}
	return false, nil
}

// isMeshProxyResponse returns true when the response was served by an Istio
// Envoy sidecar proxy. After route/traffic updates, Envoy may briefly serve
// stale content until the new configuration propagates from istiod.
func isMeshProxyResponse(resp *Response) bool {
	if resp == nil || resp.StatusCode != http.StatusOK {
		return false
	}
	server := resp.Header.Get("Server")
	return server == "istio-envoy" || strings.HasPrefix(server, "istio-envoy")
}
