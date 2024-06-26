/*
 * Copyright 2021 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package installation

import (
	pkgupgrade "knative.dev/pkg/test/upgrade"
	"knative.dev/reconciler-test/pkg/environment"
)

// LatestStable installs the latest stable eventing kafka.
func LatestStable(glob environment.GlobalEnvironment) pkgupgrade.Operation {
	return pkgupgrade.NewOperation("LatestStable", func(c pkgupgrade.Context) {
		runShellFunc("install_latest_release", c)
		cleanupTriggerv2Deployments(c, glob)
		cleanupTriggerv2ConsumerGroups(c, glob)
	})
}
