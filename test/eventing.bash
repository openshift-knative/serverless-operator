#!/usr/bin/env bash

# For SC2164
set -e

function upstream_knative_eventing_e2e {
  should_run "${FUNCNAME[0]}" || return 0

  logger.info 'Running eventing tests'

  if [[ $FULL_MESH = true ]]; then
    upstream_knative_eventing_e2e_mesh
    return $?
  fi

  export TEST_IMAGE_TEMPLATE="registry.ci.openshift.org/openshift/knative-eventing-test-{{.Name}}:${KNATIVE_EVENTING_VERSION}"

  # shellcheck disable=SC1091
  source "${KNATIVE_EVENTING_HOME}/openshift/e2e-common.sh"

  cd "${KNATIVE_EVENTING_HOME}"

  # run_e2e_tests defined in knative-eventing
  logger.info 'Starting eventing e2e tests'
  run_e2e_tests

  # run_conformance_tests defined in knative-eventing
  logger.info 'Starting eventing conformance tests'
  run_conformance_tests
}

function upstream_knative_eventing_e2e_mesh() {
  pushd "${KNATIVE_EVENTING_ISTIO_HOME}" || return $?

  echo 'diff --git a/test/e2e-common.sh b/test/e2e-common.sh
--- a/test/e2e-common.sh	(revision 62fa5877f3200ad20151a49c2103c94b8cc51a68)
+++ b/test/e2e-common.sh	(date 1690526650818)
@@ -23,6 +23,21 @@
 function run_eventing_core_tests() {
   pushd "${REPO_ROOT_DIR}"/third_party/eventing || return $?

+  echo "diff --git a/vendor/knative.dev/reconciler-test/pkg/eventshub/102-service.yaml b/vendor/knative.dev/reconciler-test/pkg/eventshub/102-service.yaml
+--- a/vendor/knative.dev/reconciler-test/pkg/eventshub/102-service.yaml	(revision 7d0c4276536808d76936049e9d01ca11fccb6589)
++++ b/vendor/knative.dev/reconciler-test/pkg/eventshub/102-service.yaml	(date 1690526603539)
+@@ -21,6 +21,7 @@
+   selector:
+     app: eventshub-{{ .name }}
+   ports:
+-    - protocol: TCP
++    - name: http
++      protocol: TCP
+       port: 80
+       targetPort: 8080" > svc.patch
+
+  git apply svc.patch
+
   BROKER_TEMPLATES="${KAFKA_BROKER_TEMPLATES}" go_test_e2e \
     -timeout=1h \
     -parallel=12 \
    ' > eventing-istio.patch

    git apply eventing-istio.patch

  ./openshift/e2e-tests.sh || return $?

  popd || return $?
}
