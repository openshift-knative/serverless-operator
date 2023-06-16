package common

import (
	mf "github.com/manifestival/manifestival"
	corev1 "k8s.io/api/core/v1"
	"knative.dev/operator/pkg/apis/operator/base"
)

// ConfigureEnvValueIfUnset sets env values via arguments if users haven't set in the CR.
func ConfigureEnvValueIfUnset(s *base.CommonSpec, deployment, container, key, value string) mf.Transformer {
	for _, o := range s.GetWorkloadOverrides() {
		if o.Name == deployment {
			for _, env := range o.Env {
				if env.Container == container {
					for _, envVar := range env.EnvVars {
						if envVar.Name == key {
							// Already set, nothing to do here.
							return nil
						}
					}
				}
			}
		}
	}
	return InjectEnvironmentIntoDeployment(deployment, container, corev1.EnvVar{Name: key, Value: value})
}
