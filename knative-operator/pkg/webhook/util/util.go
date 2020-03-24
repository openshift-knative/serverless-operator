package util

import (
	"context"
	"fmt"
	"github.com/appscode/jsonpatch"
	"github.com/coreos/go-semver/semver"
	configv1 "github.com/openshift/api/config/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// ValidateOpenShiftVersion validates the current openshift version against the minimum
// version specified in MIN_OPENSHIFT_VERSION env var
func ValidateOpenShiftVersion(ctx context.Context, cl client.Client) (bool, string, error) {
	version, present := os.LookupEnv("MIN_OPENSHIFT_VERSION")
	if !present {
		return true, "", nil
	}
	minVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, "Unable to validate version; check MIN_OPENSHIFT_VERSION env var", nil
	}

	clusterVersion := &configv1.ClusterVersion{}
	if err := cl.Get(ctx, client.ObjectKey{Name: "version"}, clusterVersion); err != nil {
		return false, "Unable to get ClusterVersion", err
	}

	current, err := semver.NewVersion(clusterVersion.Status.Desired.Version)
	if err != nil {
		return false, "Could not parse version string", err
	}

	if current.Major == 0 && current.Minor == 0 {
		return true, "CI build detected, bypassing version check", nil
	}

	if current.LessThan(*minVersion) {
		msg := fmt.Sprintf("Version constraint not fulfilled: minimum version: %s, current version: %s", minVersion.String(), current.String())
		return false, msg, nil
	}
	return true, "", nil
}

// PatchResponseFromRaw takes 2 byte arrays and returns a new response with json patch.
// The original object should be passed in as raw bytes to avoid the roundtripping problem
// described in https://github.com/kubernetes-sigs/kubebuilder/issues/510.
func PatchResponseFromRaw(original, current []byte) types.Response {
	patches, err := jsonpatch.CreatePatch(original, current)
	if err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return types.Response{
		Patches: patches,
		Response: &admissionv1beta1.AdmissionResponse{
			Allowed:   true,
			PatchType: func() *admissionv1beta1.PatchType { pt := admissionv1beta1.PatchTypeJSONPatch; return &pt }(),
		},
	}
}
