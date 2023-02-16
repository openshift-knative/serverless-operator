package kitchensink

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openshift-knative/serverless-operator/test"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/pager"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/system"
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/k8s"
	"knative.dev/reconciler-test/pkg/knative"
	"knative.dev/reconciler-test/pkg/state"
)

type FeatureWithEnvironment struct {
	Feature     *feature.Feature
	Global      environment.GlobalEnvironment
	Context     context.Context
	Environment environment.Environment
}

func (fe *FeatureWithEnvironment) CreateEnvironment() {
	ctx, env := fe.Global.Environment(
		knative.WithKnativeNamespace(system.Namespace()),
		knative.WithLoggingConfig,
		knative.WithTracingConfig,
		k8s.WithEventListener,
		environment.WithPollTimings(4*time.Second, 600*time.Second),
	)

	// Copied from reconciler-test/MagicEnvironment.
	// The Store that is inside of the Feature will be assigned to the context.
	// If no Store is set on Feature, Test will create a new store.KVStore
	// and set it on the feature and then apply it to the Context.
	if fe.Feature.State == nil {
		fe.Feature.State = &state.KVStore{}
	}
	ctx = state.ContextWith(ctx, fe.Feature.State)
	ctx = feature.ContextWith(ctx, fe.Feature)

	fe.Context = ctx
	fe.Environment = env
}

func (fe *FeatureWithEnvironment) DeleteNamespace() error {
	kube := kubeclient.Get(fe.Context)
	if err := kube.CoreV1().Namespaces().Delete(context.Background(), fe.Environment.Namespace(), metav1.DeleteOptions{}); err != nil {
		return err
	}
	waitErr := wait.PollImmediate(test.Interval, 2*test.Timeout, func() (bool, error) {
		if _, err := kube.CoreV1().Namespaces().Get(context.Background(),
			fe.Environment.Namespace(), metav1.GetOptions{}); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	if waitErr != nil {
		return fmt.Errorf("namespace %s not deleted in time: %w", fe.Environment.Namespace(), waitErr)
	}
	return nil
}

func (fe *FeatureWithEnvironment) PreUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(fe.Feature.Name, func(c pkgupgrade.Context) {
		c.T.Parallel()
		fe.CreateEnvironment()
		setups := filterStepTimings(fe.Feature.Steps, feature.Setup)
		for _, s := range setups {
			s.Fn(fe.Context, c.T)
		}
		requirements := filterStepTimings(fe.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(fe.Context, c.T)
		}
		asserts := filterStepTimings(fe.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(fe.Context, c.T)
		}
	})
}

func (fe *FeatureWithEnvironment) PostUpgrade() pkgupgrade.Operation {
	return pkgupgrade.NewOperation(fe.Feature.Name, func(c pkgupgrade.Context) {
		c.T.Parallel()
		requirements := filterStepTimings(fe.Feature.Steps, feature.Requirement)
		for _, r := range requirements {
			r.Fn(fe.Context, c.T)
		}
		asserts := filterStepTimings(fe.Feature.Steps, feature.Assert)
		for _, a := range asserts {
			a.Fn(fe.Context, c.T)
		}
		teardowns := filterStepTimings(fe.Feature.Steps, feature.Teardown)
		for _, td := range teardowns {
			td.Fn(fe.Context, c.T)
		}
		if err := fe.DeleteNamespace(); err != nil {
			c.T.Error(err)
		}
	})
}

type FeatureWithEnvironmentGroup []*FeatureWithEnvironment

func (fg FeatureWithEnvironmentGroup) PreUpgradeTests() []pkgupgrade.Operation {
	ops := make([]pkgupgrade.Operation, 0, len(fg))
	for _, ft := range fg {
		ops = append(ops, ft.PreUpgrade())
	}
	return ops
}

func (fg FeatureWithEnvironmentGroup) PostUpgradeTests() []pkgupgrade.Operation {
	ops := make([]pkgupgrade.Operation, 0, len(fg))
	for _, ft := range fg {
		ops = append(ops, ft.PostUpgrade())
	}
	return ops
}

func filterStepTimings(steps []feature.Step, timing feature.Timing) []feature.Step {
	var res []feature.Step
	for _, s := range steps {
		if s.T == timing {
			res = append(res, s)
		}
	}
	return res
}

func (fg *FeatureWithEnvironmentGroup) Split(parts int) []FeatureWithEnvironmentGroup {
	groups := make([]FeatureWithEnvironmentGroup, 0, parts)

	size := len(*fg) / parts
	var j int
	for i := 0; i < len(*fg); i += size {
		j += size
		if j+size > len(*fg) {
			// Squeeze the remainder into the last group.
			groups = append(groups, (*fg)[i:len(*fg)])
			break
		}
		groups = append(groups, (*fg)[i:j])
	}

	return groups
}

func PatchKnativeResources(ctx *test.Context) error {
	crdClient := ctx.Clients.APIExtensionClient.ApiextensionsV1().CustomResourceDefinitions()

	crdList, err := crdClient.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch crd list: %w", err)
	}

	for _, crd := range crdList.Items {
		if strings.Contains(crd.Name, "knative.dev") && !strings.Contains(crd.Name, "internal") {
			gr := schema.ParseGroupResource(crd.Name)
			if gr.Empty() {
				return fmt.Errorf("unable to parse group version: %s", crd.Name)
			}
			crd, err := crdClient.Get(context.Background(), gr.String(), metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("unable to fetch crd %s - %w", gr, err)
			}
			version := storageVersion(crd)
			if version == "" {
				return fmt.Errorf("unable to determine storage version for %s", gr)
			}
			if err := patchEmpty(ctx, gr.WithVersion(version)); err != nil {
				return err
			}
		}
	}

	return nil
}

func patchEmpty(ctx *test.Context, gvr schema.GroupVersionResource) error {
	client := ctx.Clients.Dynamic.Resource(gvr)

	listFunc := func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return client.Namespace(metav1.NamespaceAll).List(ctx, opts)
	}

	onEach := func(obj runtime.Object) error {
		item := obj.(metav1.Object)

		_, err := client.Namespace(item.GetNamespace()).
			Patch(context.Background(), item.GetName(), types.MergePatchType, []byte("{}"), metav1.PatchOptions{})

		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("unable to patch resource %s/%s (gvr: %s) - %w",
				item.GetNamespace(), item.GetName(),
				gvr, err)
		}

		return nil
	}

	pager := pager.New(listFunc)
	return pager.EachListItem(context.Background(), metav1.ListOptions{}, onEach)
}

func storageVersion(crd *apix.CustomResourceDefinition) string {
	var version string
	for _, v := range crd.Spec.Versions {
		if v.Storage {
			version = v.Name
			break
		}
	}
	return version
}
