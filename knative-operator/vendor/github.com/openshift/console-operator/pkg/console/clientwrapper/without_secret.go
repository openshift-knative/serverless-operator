package clientwrapper

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

func WithoutSecret(client kubernetes.Interface) kubernetes.Interface {
	return &clientWrapper{
		Interface: client,
		core: &coreWrapper{
			CoreV1Interface: client.CoreV1(),
			secrets:         fake.NewSimpleClientset().CoreV1(),
		},
	}
}

type clientWrapper struct {
	kubernetes.Interface
	core *coreWrapper
}

func (c *clientWrapper) CoreV1() corev1.CoreV1Interface {
	return c.core
}

func (c *clientWrapper) Secrets(namespace string) corev1.SecretInterface {
	return c.core.Secrets(namespace)
}

type coreWrapper struct {
	corev1.CoreV1Interface
	secrets corev1.SecretsGetter
}

func (c *coreWrapper) Secrets(namespace string) corev1.SecretInterface {
	return c.secrets.Secrets(namespace)
}
