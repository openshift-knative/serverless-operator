# Dockerfile to bootstrap build and test in openshift-ci

FROM openshift/origin-release:golang-__GOLANG_VERSION__

# Add kubernetes repository
ADD openshift/ci-operator/build-image/kubernetes.repo /etc/yum.repos.d/

RUN yum install -y kubectl ansible httpd-tools

RUN GO111MODULE=on go get github.com/mikefarah/yq/v3 \
  knative.dev/test-infra/kntest/cmd/kntest

# Allow runtime users to add entries to /etc/passwd
RUN chmod g+rw /etc/passwd
