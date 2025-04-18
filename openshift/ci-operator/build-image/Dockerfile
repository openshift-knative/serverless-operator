# Dockerfile to bootstrap build and test in openshift-ci
FROM registry.ci.openshift.org/openshift/release:rhel-8-release-golang-1.23-openshift-4.19

# make art yum/dnf wrapper to check in /etc/yum.repos.d/ too
ENV ART_DNF_WRAPPER_POLICY=append

# Add kubernetes repository
ADD openshift/ci-operator/build-image/kubernetes.repo /etc/yum.repos.d/

RUN yum install -y kubectl httpd-tools

RUN GOFLAGS='' go install github.com/mikefarah/yq/v3@latest
RUN GOFLAGS='' go install knative.dev/test-infra/tools/kntest/cmd/kntest@latest
RUN GOFLAGS='' go install -tags="exclude_graphdriver_btrfs containers_image_openpgp" github.com/containers/skopeo/cmd/skopeo@v1.17.0
RUN rm -rf $GOPATH/.cache

# Allow runtime users to add entries to /etc/passwd
RUN chmod g+rw /etc/passwd

RUN yum install -y https://rpm.nodesource.com/pub_20.x/el/8/x86_64/nodesource-release-el8-1.noarch.rpm
RUN yum module disable -y nodejs
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
