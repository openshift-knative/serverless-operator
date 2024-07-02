# Use the tools image to to install kubectl/oc
FROM registry.ci.openshift.org/ocp/4.16:tools as tools

# Dockerfile to bootstrap build and test in openshift-ci
FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-__GOLANG_VERSION__-openshift-4.16

COPY --from=tools /usr/bin/oc /usr/bin/
RUN ln -s /usr/bin/oc /usr/bin/kubectl

RUN GOFLAGS='' go install github.com/mikefarah/yq/v3@latest
RUN GOFLAGS='' go install knative.dev/test-infra/tools/kntest/cmd/kntest@latest
RUN rm -rf $GOPATH/.cache

# Allow runtime users to add entries to /etc/passwd
RUN chmod g+rw /etc/passwd

RUN yum install -y https://rpm.nodesource.com/pub___NODEJS_VERSION__/el/8/x86_64/nodesource-release-el8-1.noarch.rpm
RUN yum module disable -y nodejs
RUN yum install -y \
  httpd-tools \
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
