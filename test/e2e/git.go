package e2e

import (
	"fmt"
	"github.com/openshift-knative/serverless-operator/test"
	"os"
	"path"
	"testing"
)

type repository struct {
	source    string
	scmserver string
	name      []string
	branch    string
	version   string
}

func checkout(t *testing.T, ctx *test.Context, repo repository, gopath string) string {
	target := target(repo, gopath)
	parent := path.Dir(target)

	wd, err := os.Getwd()
	ensureNoError(t, err)
	ctx.AddToCleanup(func() error {
		return os.Chdir(wd)
	})

	t.Run("git checkout", func(t *testing.T) {
		ensureNoError(t, os.MkdirAll(parent, os.ModePerm))
		command := fmt.Sprintf(
			"git clone --branch '%s' --single-branch %s %s",
			repo.branch,
			repo.source,
			target)
		execute(command, t)
	})

	return target
}

func target(repo repository, gopath string) string {
	parent := path.Join(gopath, "src", repo.scmserver)
	target := parent
	for _, name := range repo.name {
		target = path.Join(target, name)
	}
	return target
}
