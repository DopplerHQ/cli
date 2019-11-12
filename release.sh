#!/bin/bash

set -e

function finish {
  # delete google creds from filesystem
  rm -f "$GOOGLE_APPLICATION_CREDENTIALS"

  # delete docker creds
  set +e
  docker logout $DOCKER_REGISTRY
  set -e
  rm -rf "$DOCKER_CONFIG"
}
trap finish EXIT

if [ ! -z "$(git status --porcelain)" ]; then
  echo "The git workspace must be clean"
  exit 1
fi

if [ $# -eq 0 ]; then
  echo "You must specify a version"
  exit 1
fi

TAGNAME=$1
if [ "${TAGNAME:0:1}" != "v" ]; then
  echo "Version is incorrect; must match format vX.Y.Z"
  exit 1
fi

echo "Using version $TAGNAME"

# get git in order
git push
git tag -a "$TAGNAME" -m "$TAGNAME"
git push origin "$TAGNAME"

# save google creds to filesystem
echo "$GOOGLE_CREDS" > "$GOOGLE_APPLICATION_CREDENTIALS"
# config will be saved to location explicitly specified in $DOCKER_CONFIG (set by Doppler)
echo $DOCKER_HUB_TOKEN | docker login -u $DOCKER_HUB_USER --password-stdin $DOCKER_REGISTRY
echo $GOOGLE_CREDS | docker login -u $GCR_USER --password-stdin $GCR_REGISTRY
goreleaser release --rm-dist

scripts/publish-deb.sh
scripts/publish-rpm.sh
