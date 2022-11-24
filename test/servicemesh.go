package test

import (
	"context"

	socommon "github.com/openshift-knative/serverless-operator/pkg/common"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func ServiceMeshControlPlaneV2(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "maistra.io/v2",
			"kind":       "ServiceMeshControlPlane",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"version": "v2.1",
			},
		},
	}
}

func ServiceMeshMemberRollV1(name, namespace string, members ...string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "maistra.io/v1",
			"kind":       "ServiceMeshMemberRoll",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"members": members,
			},
		},
	}
}

func serviceMeshControlPlaneV2Schema() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "maistra.io",
		Version:  "v2",
		Resource: "servicemeshcontrolplanes",
	}
}

func CreateServiceMeshControlPlaneV2(ctx *Context, smcp *unstructured.Unstructured) *unstructured.Unstructured {
	// When cleaning-up SMCP, wait until it doesn't exist, as it takes a while, which would break subsequent tests
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Waiting for ServiceMeshControlPlane %q to not exist", smcp.GetName())
		_, err := WaitForUnstructuredState(ctx, serviceMeshControlPlaneV2Schema(), smcp.GetName(), smcp.GetNamespace(), DoesUnstructuredNotExist)
		return err
	})
	return CreateUnstructured(ctx, serviceMeshControlPlaneV2Schema(), smcp)
}

func WaitForServiceMeshControlPlaneReady(ctx *Context, name, namespace string) {
	// We use v2 schema for Readiness even if we install a "v.1.1" (v1 schema doesn't have "conditions")
	_, err := WaitForUnstructuredState(ctx, serviceMeshControlPlaneV2Schema(), name, namespace, IsUnstructuredReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ServiceMeshControlPlane readiness: %v", err)
	}
}

func GetServiceMeshControlPlaneVersion(ctx *Context, name, namespace string) (string, bool, error) {
	smcp, err := ctx.Clients.Dynamic.Resource(serviceMeshControlPlaneV2Schema()).Namespace(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		ctx.T.Fatalf("Error getting SMCP %s: %v", name, err)
	}

	return unstructured.NestedString(smcp.Object, "spec", "version")
}

func CreateServiceMeshMemberRollV1(ctx *Context, smmr *unstructured.Unstructured) *unstructured.Unstructured {
	smmrGvr := schema.GroupVersionResource{
		Group:    "maistra.io",
		Version:  "v1",
		Resource: "servicemeshmemberrolls",
	}

	return CreateUnstructured(ctx, smmrGvr, smmr)
}

func AllowFromServingSystemNamespaceNetworkPolicy(namespace string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-from-system-namespace",
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									socommon.ServerlessCommonLabelKey: socommon.ServerlessCommonLabelValue,
								},
							},
						},
					},
				},
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
		},
	}
}

func CreateNetworkPolicy(ctx *Context, networkPolicy *networkingv1.NetworkPolicy) *networkingv1.NetworkPolicy {
	createdNetworkPolicy, err := ctx.Clients.Kube.NetworkingV1().NetworkPolicies(networkPolicy.Namespace).Create(context.Background(), networkPolicy, metav1.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating NetworkPolicy %s: %v", networkPolicy.GetName(), err)
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up NetworkPolicy %s", createdNetworkPolicy.GetName())
		return ctx.Clients.Kube.NetworkingV1().NetworkPolicies(createdNetworkPolicy.Namespace).Delete(context.Background(), createdNetworkPolicy.GetName(), metav1.DeleteOptions{})
	})

	return createdNetworkPolicy
}
