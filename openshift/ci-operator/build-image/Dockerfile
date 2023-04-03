# Dockerfile to bootstrap build and test in openshift-ci

FROM registry.ci.openshift.org/openshift/release:golang-1.18

# Add kubernetes repository
ADD openshift/ci-operator/build-image/kubernetes.repo /etc/yum.repos.d/

RUN yum install -y kubectl httpd-tools

# Create a temp directory for the go_run() function that is writable by runtime users
ENV GORUN_PATH=/tmp/gorun
RUN mkdir -p /tmp/gorun && chmod g+rw /tmp/gorun

RUN GOFLAGS='' go install github.com/mikefarah/yq/v3@latest
RUN GOFLAGS='' go install knative.dev/test-infra/tools/kntest/cmd/kntest@latest

# Allow runtime users to add entries to /etc/passwd
RUN chmod g+rw /etc/passwd

RUN yum install -y https://rpm.nodesource.com/pub_16.x/el/7/x86_64/nodesource-release-el7-1.noarch.rpm
RUN yum install -y \
  gcc-c++ \
  make \
  nodejs \
  xorg-x11-server-Xvfb \
  gtk2-devel \
  gtk3-devel \
  libnotify-devel \
  GConf2 \
  nss \
  libXScrnSaver \
  alsa-lib
