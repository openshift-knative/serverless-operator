package test

import (
	"fmt"
	"strings"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func Service(name, namespace, image string) *servingv1.Service {
	s := &servingv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
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

func WithServiceReady(ctx *Context, name, namespace, image string) (*servingv1.Service, error) {
	service, err := CreateService(ctx, name, namespace, image)
	if err != nil {
		return nil, err
	}
	service, err = WaitForServiceState(ctx, service.Name, service.Namespace, IsServiceReady)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func CreateService(ctx *Context, name, namespace, image string) (*servingv1.Service, error) {
	service, err := ctx.Clients.Serving.ServingV1().Services(namespace).Create(Service(name, namespace, image))
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Knative Service '%s/%s'", service.Namespace, service.Name)
		return ctx.Clients.Serving.ServingV1().Services(namespace).Delete(service.Name, &metav1.DeleteOptions{})
	})
	return service, nil
}

func WaitForControllerEnvironment(ctx *Context, ns, envName, envValue string) error {
	return wait.PollImmediate(Interval, 10*time.Minute, func() (bool, error) {
		pods, err := ctx.Clients.Kube.CoreV1().Pods(ns).List(metav1.ListOptions{
			LabelSelector: "app=controller",
		})
		if apierrs.IsUnauthorized(err) {
			// These errors happen when resetting the proxy value. Just retry.
			return false, nil
		} else if err != nil {
			return false, err
		}
		for _, pod := range pods.Items {
			if !isPodReady(pod) {
				return false, nil
			}
			for _, container := range pod.Spec.Containers {
				for _, e := range container.Env {
					if e.Name == envName && e.Value == envValue {
						return true, nil
					}
				}

			}
		}
		return false, nil
	})
}

func CheckDeploymentScale(ctx *Context, ns, name string, scale int) error {
	d, err := ctx.Clients.Kube.AppsV1().Deployments(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if d.Status.ReadyReplicas != int32(scale) {
		return fmt.Errorf("unexpected number of replicas: %d, expected: %d", d.Status.ReadyReplicas, scale)
	}
	return nil
}

func isPodReady(pod corev1.Pod) bool {
	if pod.DeletionTimestamp != nil || pod.Status.PodIP == "" {
		return false
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func WaitForServiceState(ctx *Context, name, namespace string, inState func(s *servingv1.Service, err error) (bool, error)) (*servingv1.Service, error) {
	var (
		lastState *servingv1.Service
		err       error
	)
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Serving.ServingV1().Services(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "knative service %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func WaitForOperatorDepsDeleted(ctx *Context) error {
	serverlessDependencies := []string{"knative-openshift-ingress",
		"knative-serving-operator", "knative-eventing-operator", "knative-openshift"}

	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		existingDeployments, err := ctx.Clients.Kube.AppsV1().Deployments(OperatorsNamespace).List(metav1.ListOptions{})
		for _, deployment := range existingDeployments.Items {
			for _, serverlessDep := range serverlessDependencies {
				if strings.Contains(deployment.Name, serverlessDep) {
					return false, err
				}
			}
		}
		return true, err
	})

	if waitErr != nil {
		return errors.Wrapf(waitErr, "serverless operator dependencies not deleted in time")
	}
	return nil
}

func IsServiceReady(s *servingv1.Service, err error) (bool, error) {
	return s.Generation == s.Status.ObservedGeneration && s.Status.IsReady() && s.Status.URL != nil && s.Status.URL.Host != "", err
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
						"app": name,
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

	if _, err := ctx.Clients.Kube.AppsV1().Deployments(namespace).Create(deployment); err != nil {
		return err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up Deployment '%s/%s'", deployment.Namespace, deployment.Name)
		return ctx.Clients.Kube.AppsV1().Deployments(namespace).Delete(deployment.Name, &metav1.DeleteOptions{})
	})

	return nil
}

func CreateKubeService(ctx *Context, name, namespace string) (*corev1.Service, error) {
	kubeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 80,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8080,
					},
				},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}

	svc, err := ctx.Clients.Kube.CoreV1().Services(namespace).Create(kubeService)
	if err != nil {
		return nil, err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up K8s Service '%s/%s'", kubeService.Namespace, kubeService.Name)
		return ctx.Clients.Serving.ServingV1beta1().Services(namespace).Delete(svc.Name, &metav1.DeleteOptions{})
	})

	return svc, nil
}

func WithRouteForServiceReady(ctx *Context, serviceName, namespace string) (*routev1.Route, error) {
	r := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
		},
	}

	route, err := ctx.Clients.Route.Routes(namespace).Create(r)
	if err != nil {
		return nil, err
	}

	ctx.AddToCleanup(func() error {
		ctx.T.Logf("Cleaning up OCP Route '%s/%s'", r.Namespace, r.Name)
		return ctx.Clients.Route.Routes(namespace).Delete(route.Name, &metav1.DeleteOptions{})
	})

	return WaitForRouteState(ctx, route.Name, route.Namespace, RouteHasHost)
}

func WaitForRouteState(ctx *Context, name, namespace string, inState func(s *routev1.Route, err error) (bool, error)) (*routev1.Route, error) {
	var (
		lastState *routev1.Route
		err       error
	)
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = ctx.Clients.Route.Routes(namespace).Get(name, metav1.GetOptions{})
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "OpenShift Route %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func RouteHasHost(r *routev1.Route, err error) (bool, error) {
	return len(r.Status.Ingress) != 0 && len(r.Status.Ingress[0].Conditions) != 0 &&
		r.Status.Ingress[0].Conditions[0].Type == routev1.RouteAdmitted &&
		r.Status.Ingress[0].Conditions[0].Status == corev1.ConditionTrue, nil
}

func int32Ptr(i int32) *int32 { return &i }
