package common

import (
	"strings"
)

const imagePrefix = "IMAGE_"

func ImageMapFromEnvironment(env []string) map[string]string {
	overrideMap := map[string]string{}

	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		if strings.HasPrefix(pair[0], imagePrefix) {
			if pair[1] == "" {
				continue
			}

			// convert
			// "IMAGE_container=docker.io/foo"
			// "IMAGE_deployment__container=docker.io/foo2"
			// "IMAGE_env_var=docker.io/foo3"
			// "IMAGE_deployment__env_var=docker.io/foo4"
			// to
			// container: docker.io/foo
			// deployment/container: docker.io/foo2
			// env_var: docker.io/foo3
			// deployment/env_var: docker.io/foo4
			name := strings.TrimPrefix(pair[0], imagePrefix)
			name = strings.Replace(name, "__", "/", 1)
			overrideMap[name] = pair[1]
		}
	}
	return overrideMap
}
