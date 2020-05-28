#!/usr/bin/env bash

KOURIER_VERSION=release-0.15
DOWNLOAD_URL=https://github.com/knative/net-kourier/releases/download/v0.15.0/kourier.yaml

if [ -f "kourier-${KOURIER_VERSION}.yaml" ]; then
  echo "kourier-${KOURIER_VERSION}.yaml already exists. Please remove it."
  echo -e "Run:\n   rm kourier-${KOURIER_VERSION}.yaml"
  exit 1
fi

wget --no-check-certificate $DOWNLOAD_URL -O kourier-${KOURIER_VERSION}.yaml
if [ $? != 0 ]; then
  echo "Failed to download kourier yaml"
  exit 1
fi

cp kourier-${KOURIER_VERSION}.yaml kourier-${KOURIER_VERSION}-debug.yaml

if [ -L "kourier-latest.yaml" ]; then
  unlink kourier-latest.yaml
fi
if [ -L "kourier-latest-debug.yaml" ]; then
  unlink kourier-latest-debug.yaml
fi

ln -s kourier-${KOURIER_VERSION}.yaml       kourier-latest.yaml
ln -s kourier-${KOURIER_VERSION}-debug.yaml kourier-latest-debug.yaml

# Apply debug log enable path to -debug.yaml only
#patch kourier-${KOURIER_VERSION}-debug.yaml debug-log.patch
