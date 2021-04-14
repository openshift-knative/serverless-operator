package common

import (
	v1 "k8s.io/api/core/v1"
)

func AppendUnique(orgEnv []v1.EnvVar, key, value string) []v1.EnvVar {
	// Set the value if the key is already present.
	for i := range orgEnv {
		if orgEnv[i].Name == key {
			orgEnv[i].Value = value
			return orgEnv
		}
	}
	// If not, append a key-value pair.
	return append(orgEnv, v1.EnvVar{
		Name:  key,
		Value: value,
	})
}
