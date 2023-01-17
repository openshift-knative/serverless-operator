package test

import (
	"context"
	"fmt"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	servingv1alpha1 "knative.dev/serving/pkg/apis/serving/v1alpha1"
)

type ServiceCfgFunc func(*servingv1.Service)

func Service(name, namespace, image string, annotations map[string]string) *servingv1.Service {
	s := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: annotations,
					},
					Spec: servingv1.RevisionSpec{
						PodSpec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Image: image,
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceName("cpu"): resource.MustParse("25m"),
									},
								},
							}},
						},
					},
				},
			},
		},
	}
	return s
}

func WithServiceReadyOrFail(ctx *Context, service *servingv1.Service) *servingv1.Service {
	service, err := ctx.Clients.Serving.ServingV1().Services(service.Namespace).Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		ctx.T.Fatalf("Error creating ksvc: %v", err)
	}

	// Let the ksvc be deleted after test
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
		return ctx.Clients.Serving.ServingV1().Services(service.Namespace).Delete(context.Background(), service.Name, metav1.DeleteOptions{})
	})

	service, err = WaitForServiceState(ctx, service.Name, service.Namespace, IsServiceReady)
	if err != nil {
		ctx.T.Fatalf("Error waiting for ksvc readiness: %v", err)
	}

	return service
}

func WithServiceReady(ctx *Context, name, namespace, image string, cfgFuncs ...ServiceCfgFunc) (*servingv1.Service, error) {
	service, err := CreateService(ctx, name, namespace, image, cfgFuncs...)
	if err != nil {
		return nil, err
	}

	service, err = WaitForServiceState(ctx, service.Name, service.Namespace, IsServiceReady)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func CreateService(ctx *Context, name, namespace, image string, cfgFuncs ...ServiceCfgFunc) (*servingv1.Service, error) {
	service := Service(name, namespace, image, nil)
	for _, f := range cfgFuncs {
		f(service)
	}

	service, err := ctx.Clients.Serving.ServingV1().Services(namespace).Create(context.Background(), service, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
		return ctx.Clients.Serving.ServingV1().Services(namespace).Delete(context.Background(), service.Name, metav1.DeleteOptions{})
	})
	return service, nil
}

func CheckDeploymentScale(ctx *Context, ns, name string, scale int) error {
	d, err := ctx.Clients.Kube.AppsV1().Deployments(ns).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if *d.Spec.Replicas != int32(scale) {
		return fmt.Errorf("unexpected number of replicas: %d, expected: %d", *d.Spec.Replicas, scale)
	}
	return nil
}

func WaitForServiceState(ctx *Context, name, namespace string, inState func(s *servingv1.Service, err error) (bool, error)) (*servingv1.Service, error) {
	var (
		lastState *servingv1.Service
		err       error
	)
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Serving.ServingV1().Services(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knative service %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func WaitForReadyServices(ctx *Context, namespace string) error {
	services, err := ctx.Clients.Serving.ServingV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, svc := range services.Items {
		_, err = WaitForServiceState(ctx, svc.Name, namespace, IsServiceReady)
		if err != nil {
			return err
		}
	}
	return nil
}

func WaitForDomainMappingState(ctx *Context, name, namespace string, inState func(dm *servingv1alpha1.DomainMapping, err error) (bool, error)) (*servingv1alpha1.DomainMapping, error) {
	var (
		lastState *servingv1alpha1.DomainMapping
		err       error
	)
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Serving.ServingV1alpha1().DomainMappings(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("knative service %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func IsServiceReady(s *servingv1.Service, err error) (bool, error) {
	return s.IsReady() && s.Status.URL != nil && s.Status.URL.Host != "", err
}

func IsDomainMappingReady(dm *servingv1alpha1.DomainMapping, err error) (bool, error) {
	return dm.IsReady() && dm.Status.URL != nil && dm.Status.URL.Host != "", err
}

func CreateDeployment(ctx *Context, name, namespace, image string) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                     name,
						"maistra.io/expose-route": "true",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	if _, err := ctx.Clients.Kube.AppsV1().Deployments(namespace).Create(context.Background(), deployment, metav1.CreateOptions{}); err != nil {
		return err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Deployment '%s/%s'", deployment.Namespace, deployment.Name)
		return ctx.Clients.Kube.AppsV1().Deployments(namespace).Delete(context.Background(), deployment.Name, metav1.DeleteOptions{})
	})

	return nil
}

func WaitForRouteState(ctx *Context, name, namespace string, inState func(s *routev1.Route, err error) (bool, error)) (*routev1.Route, error) {
	var (
		lastState *routev1.Route
		err       error
	)
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Route.Routes(namespace).Get(context.Background(), name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, fmt.Errorf("openShift route %s is not in desired state, got: %+v: %w", name, lastState, waitErr)
	}
	return lastState, nil
}

func int32Ptr(i int32) *int32 { return &i }

func WaitForServerlessOperatorsDeleted(ctx *Context) error {
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		existingDeployments, err := ctx.Clients.Kube.AppsV1().Deployments(OperatorsNamespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return true, err
		}
		for _, deployment := range existingDeployments.Items {
			if strings.Contains(deployment.Name, "knative") {
				return false, nil
			}
		}
		return true, err
	})

	if waitErr != nil {
		return fmt.Errorf("serverless operator dependencies not deleted in time: %w", waitErr)
	}
	return nil
}
