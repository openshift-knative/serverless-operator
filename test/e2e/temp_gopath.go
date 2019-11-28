package e2e

import (
	"fmt"
	"github.com/openshift-knative/serverless-operator/test"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

type repositoryInTempGopath struct {
	gopath string
	repo   repository
}

func createTemporaryGopath(t *testing.T, ctx *test.Context) string {
	gopath, err := ioutil.TempDir("", "test-gopath")
	ensureNoError(t, err)
	ctx.AddToCleanup(func() error {
		if ctx.T.Failed() && runsOnOpenshiftCI() {
			ctx.T.Logf("Tests have failed, so let's leave temp GOPATH for inspection: %s", gopath)
			return nil
		}
		return os.RemoveAll(gopath)
	})

	t.Run("create temporary gopath", func(t *testing.T) {
		ensureNoError(t, os.MkdirAll(path.Join(gopath, "bin"), os.ModePerm))
		execute(fmt.Sprintf("cp -rv \"$(go env GOPATH)/bin\" '%s'", gopath), t)
	})

	return gopath
}

func (r repositoryInTempGopath) test(t *testing.T, commandSupplier func(dir string) string) {
	dir := target(r.repo, r.gopath)
	ensureNoError(t, os.Chdir(dir))
	command := commandSupplier(dir)
	streamExecutionAndFailLate(command, t)
}
