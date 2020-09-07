#!/bin/bash

set -e

function cleanup {
  # delete docker creds
  set +e
  docker logout "$DOCKER_REGISTRY"
  docker logout "$GCR_REGISTRY"
  set -e
  rm -rf "$DOCKER_CONFIG"
}
trap cleanup EXIT

echo "Using Docker config $DOCKER_CONFIG"

# config will be saved to location explicitly specified in $DOCKER_CONFIG (set by Doppler)
echo "$DOCKER_HUB_TOKEN" | docker login -u "$DOCKER_HUB_USER" --password-stdin "$DOCKER_REGISTRY"
echo "$GOOGLE_CREDS" | docker login -u "$GCR_USER" --password-stdin "$GCR_REGISTRY"

# pull in latest docker images
docker pull alpine
docker pull node:lts-alpine
docker pull python:3-alpine
docker pull ruby:2-alpine

GOOGLE_APPLICATION_CREDENTIALS=<(echo "$GOOGLE_CREDS") goreleaser release --rm-dist
