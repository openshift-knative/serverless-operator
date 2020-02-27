#!/usr/bin/env bash

KOURIER_VERSION=v0.3.10
DOWNLOAD_URL=https://raw.githubusercontent.com/3scale/kourier/${KOURIER_VERSION}/deploy/kourier-knative.yaml

if [ -f "kourier-${KOURIER_VERSION}.yaml" ]; then
  echo "kourier-${KOURIER_VERSION}.yaml already exists"
  exit 1
fi

wget --no-check-certificate $DOWNLOAD_URL -O kourier-${KOURIER_VERSION}.yaml
if [ $? != 0 ]; then
  echo "Failed to download kourier yaml"
  exit 1
fi

if [ -L "kourier-latest.yaml" ]; then
  unlink kourier-latest.yaml
fi

ln -s kourier-${KOURIER_VERSION}.yaml kourier-latest.yaml

patch kourier-${KOURIER_VERSION}.yaml proxyv2-image.patch
