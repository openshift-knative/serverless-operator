#!/usr/bin/env bash

ORG="${ORG:-"openshift-knative"}"
IMAGE=$(basename $(pwd))
BRANCH=$(git rev-parse --abbrev-ref HEAD)
eval VERSION=v$(grep Version version/version.go | awk '{print $3}')

function tag {
  local url="https://quay.io/api/v1/repository/$ORG/$IMAGE/tag/"
  local i=$(curl -s $url | jq '."tags"[]["name"]' | grep "$VERSION-$BRANCH-" | sed 's/"//g' | awk -F'-' '{print $3}' | head -1)
  printf -- "$VERSION-$BRANCH-%02d" $((++i))
}

TAG=${TAG:-$(tag)}
echo Tagging $ORG/$IMAGE with $TAG
read -p "Enter to continue or Ctrl-C to exit: "

set -ex

pushd $(dirname "$0")/..
operator-sdk build "quay.io/$ORG/$IMAGE:$TAG"
docker push "quay.io/$ORG/$IMAGE:$TAG"
git tag -f "$ORG/$TAG"
git push --tags --force
popd

set +x
echo "Don't forget to update your manifest image tags to '$TAG'"
