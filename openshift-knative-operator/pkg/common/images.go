package common

import (
	"strings"
)

const imagePrefix = "IMAGE_"

// ImageMapFromEnvironment generates a map of deployment/container to image from
// the passed environment variables.
func ImageMapFromEnvironment(env []string) map[string]string {
	overrideMap := map[string]string{}

	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], imagePrefix) {
			if pair[1] == "" {
				continue
			}

			/*
				converts:
				IMAGE_container=foo             -> container: foo
				IMAGE_deployment__container=foo -> deployment/container: foo
				IMAGE_env_var=foo               -> env_var: foo
				IMAGE_deployment__env_var=foo   -> deployment/env_var: foo
			*/
			name := strings.TrimPrefix(pair[0], imagePrefix)
			name = strings.Replace(name, "__", "/", 1)
			overrideMap[name] = pair[1]
		}
	}
	return overrideMap
}
