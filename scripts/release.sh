#!/bin/bash

set -e

function cleanup {
  # delete docker creds
  set +e
  docker logout $DOCKER_REGISTRY
  docker logout $GCR_REGISTRY
  set -e
  rm -rf "$DOCKER_CONFIG"
}
trap cleanup EXIT

echo "Using Docker config $DOCKER_CONFIG"

# save google creds to filesystem
# config will be saved to location explicitly specified in $DOCKER_CONFIG (set by Doppler)
echo $DOCKER_HUB_TOKEN | docker login -u $DOCKER_HUB_USER --password-stdin $DOCKER_REGISTRY
echo $GOOGLE_CREDS | docker login -u $GCR_USER --password-stdin $GCR_REGISTRY

GOOGLE_APPLICATION_CREDENTIALS="<(echo $GOOGLE_CREDS)" goreleaser release --rm-dist
scripts/publish-deb.sh
scripts/publish-rpm.sh
