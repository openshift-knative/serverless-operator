// +build e2e

package e2e

import (
	"fmt"
	"github.com/openshift-knative/serverless-operator/test"
	"os"
	"strings"
	"testing"
)

func TestOpenshiftKnativeServing(t *testing.T) {
	version := "v0.9.0"
	repo := repository{
		source:    "https://github.com/openshift/knative-serving.git",
		scmserver: "knative.dev",
		name:      []string{"serving"},
		branch:    "release-" + version,
		version:   version,
	}
	ctx := test.SetupClusterAdmin(t)

	deployServerlessOperator(t, ctx)
	deployKnativeServingResource(t, ctx)

	gopath := createTemporaryGopath(t, ctx)
	target := checkout(t, ctx, repo, gopath)
	hackOnOpenshiftKnativeServingRepo(t, target)
	createTestResourcesForOpenshiftKnativeServing(t, target, ctx)

	r := repositoryInTempGopath{
		gopath: gopath,
		repo:   repo,
	}

	t.Run("run e2e tests", func(t *testing.T) {
		testOpenshiftKnativeServing(r, t, "./test/e2e")
	})
	t.Run("run conformance-runtime tests", func(t *testing.T) {
		testOpenshiftKnativeServing(r, t, "./test/conformance/runtime/...")
	})
	t.Run("run conformance-api tests", func(t *testing.T) {
		testOpenshiftKnativeServing(r, t, "./test/conformance/api/...")
	})

	removeKnativeServingResource(t, ctx)
	removeServerlessOperator(t, ctx)
}

func testOpenshiftKnativeServing(r repositoryInTempGopath, t *testing.T, scope string) {
	r.test(t, func(dir string) string {
		envs := fmt.Sprintf(
			"env GATEWAY_NAMESPACE_OVERRIDE='knative-serving-ingress' GOPATH='%s'",
			r.gopath)
		imageTemplate := fmt.Sprintf(
			"registry.svc.ci.openshift.org/openshift/knative-%s:knative-serving-test-{{.Name}}",
			r.repo.version)
		kubeconfigs := strings.Split(test.Flags.Kubeconfigs, ",")
		kubeconfig := kubeconfigs[0]
		return fmt.Sprintf("%s go test -v "+
			"-tags=e2e -count=1 -timeout=30m -parallel=3 %s "+
			"--resolvabledomain --kubeconfig '%s' "+
			"--imagetemplate '%s'",
			envs, scope, kubeconfig, imageTemplate)
	})
}

// namespaces, configMaps, secrets
func createTestResourcesForOpenshiftKnativeServing(t *testing.T, target string, ctx *test.Context) {
	t.Run("create test resources for openshift-knativeserving", func(t *testing.T) {
		ensureNoError(t, os.Chdir(target))
		execute("oc apply -f test/config", t)
		execute("oc adm policy add-scc-to-user privileged -z default -n serving-tests", t)
		execute("oc adm policy add-scc-to-user privileged -z default -n serving-tests-alt", t)
		execute("oc adm policy add-scc-to-user anyuid -z default -n serving-tests", t)
	})
}

func hackOnOpenshiftKnativeServingRepo(t *testing.T, target string) {
	t.Run("hack on openshift-knativeserving repo", func(t *testing.T) {
		ensureNoError(t, os.Chdir(target))
		execute("rm -vf test/config/100-istio-default-domain.yaml", t)
	})
}
