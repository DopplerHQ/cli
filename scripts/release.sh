#!/bin/bash

set -e

function cleanup {
  # delete google creds from filesystem
  rm -f "$GOOGLE_APPLICATION_CREDENTIALS"

  # delete docker creds
  set +e
  docker logout $DOCKER_REGISTRY
  docker logout $GCR_REGISTRY
  set -e
  rm -rf "$DOCKER_CONFIG"
}
trap cleanup EXIT

# save google creds to filesystem
echo "$GOOGLE_CREDS" > "$GOOGLE_APPLICATION_CREDENTIALS"
# config will be saved to location explicitly specified in $DOCKER_CONFIG (set by Doppler)
echo $DOCKER_HUB_TOKEN | docker login -u $DOCKER_HUB_USER --password-stdin $DOCKER_REGISTRY
echo $GOOGLE_CREDS | docker login -u $GCR_USER --password-stdin $GCR_REGISTRY

goreleaser release --rm-dist
scripts/publish-deb.sh
scripts/publish-rpm.sh
