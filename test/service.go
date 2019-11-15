package test

import (
	"strings"

	servingv1beta1 "github.com/knative/serving/pkg/apis/serving/v1beta1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
)

func Service(name, namespace, image string) *servingv1beta1.Service {
	s := &servingv1beta1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: servingv1beta1.ServiceSpec{
			ConfigurationSpec: servingv1beta1.ConfigurationSpec{
				Template: servingv1beta1.RevisionTemplateSpec{
					Spec: servingv1beta1.RevisionSpec{
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

func WithServiceReady(ctx *Context, name, namespace, image string) (*servingv1beta1.Service, error) {
	service, err := CreateService(ctx, name, namespace, image)
	if err != nil {
		return nil, err
	}
	_, err = WaitForServiceState(ctx, service.Name, service.Namespace, IsServiceReady)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func CreateService(ctx *Context, name, namespace, image string) (*servingv1beta1.Service, error) {
	service, err := ctx.Clients.Serving.ServingV1beta1().Services(namespace).Create(Service(name, namespace, image))
	if err != nil {
		return nil, err
	}
	ctx.AddToCleanup(func() error {
		return DeleteService(ctx, service.Name, namespace)
	})
	return service, nil
}

func GetService(ctx *Context, name, namespace string) (*servingv1beta1.Service, error) {
	return ctx.Clients.Serving.ServingV1beta1().Services(namespace).Get(name, metav1.GetOptions{})
}

func ListServices(ctx *Context, namespace string) (*servingv1beta1.ServiceList, error) {
	return ctx.Clients.Serving.ServingV1beta1().Services(namespace).List(metav1.ListOptions{})
}

func DeleteService(ctx *Context, name, namespace string) error {
	return ctx.Clients.Serving.ServingV1beta1().Services(namespace).Delete(name, &metav1.DeleteOptions{})
}

func WaitForServiceState(ctx *Context, name, namespace string, inState func(s *servingv1beta1.Service, err error) (bool, error)) (*servingv1beta1.Service, error) {
	var lastState *servingv1beta1.Service
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = GetService(ctx, name, namespace)
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "knative service %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func WaitForOperatorDepsDeleted(ctx *Context) error {
	serverlessDependencies := []string{"knative-openshift-ingress",
		"knative-serving-operator"}

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

func IsServiceReady(s *servingv1beta1.Service, err error) (bool, error) {
	return s.Generation == s.Status.ObservedGeneration && s.Status.IsReady(), err
}

func CreateKubeService(ctx *Context, name, namespace, image string) (*corev1.Service, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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

	_, err := ctx.Clients.Kube.AppsV1().Deployments(namespace).Create(deployment)
	if err != nil {
		return nil, err
	}

	ctx.AddToCleanup(func() error {
		return DeleteDeployment(ctx, deployment.Name, namespace)
	})

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
		return DeleteService(ctx, svc.Name, namespace)
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
		return DeleteRoute(ctx, route.Name, namespace)
	})

	route, err = WaitForRouteState(ctx, route.Name, route.Namespace, RouteHasHost)
	if err != nil {
		return nil, err
	}

	return route, nil
}

func WaitForRouteState(ctx *Context, name, namespace string, inState func(s *routev1.Route, err error) (bool, error)) (*routev1.Route, error) {
	var lastState *routev1.Route
	var err error
	waitErr := wait.PollImmediate(Interval, Timeout, func() (bool, error) {
		lastState, err = GetRoute(ctx, name, namespace)
		return inState(lastState, err)
	})

	if waitErr != nil {
		return lastState, errors.Wrapf(waitErr, "OpenShift Route %s is not in desired state, got: %+v", name, lastState)
	}
	return lastState, nil
}

func GetRoute(ctx *Context, name, namespace string) (*routev1.Route, error) {
	return ctx.Clients.Route.Routes(namespace).Get(name, metav1.GetOptions{})
}

func RouteHasHost(r *routev1.Route, err error) (bool, error) {
	return len(r.Status.Ingress) != 0 && r.Status.Ingress[0].Host != "", nil
}

func DeleteKubeService(ctx *Context, name, namespace string) error {
	if err := ctx.Clients.Kube.CoreV1().Services(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteDeployment(ctx *Context, name, namespace string) error {
	if err := ctx.Clients.Kube.AppsV1().Deployments(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func DeleteRoute(ctx *Context, name, namespace string) error {
	if err := ctx.Clients.Route.Routes(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
