#!/bin/bash

set -eu -o pipefail -o functrace

# make sure docker daemon is running
docker ps > /dev/null 2>&1 || (echo "Docker daemon must be running" && exit 1)

if [ ! -z "$(git status --porcelain)" ]; then
  echo "The git workspace must be clean"
  exit 1
fi

if [ $# -eq 0 ]; then
  echo "You must specify a release type or version: major|minor|patch|v1.0.0"
  exit 1
fi

RELEASE_TYPE=$1
PREV_VERSION=$(git describe --abbrev=0)

if [ "${RELEASE_TYPE:0:1}" == "v" ]; then
  VERSION="${RELEASE_TYPE:1}"
else
  reltype=""
  if [ "$RELEASE_TYPE" == "major" ]; then
    reltype="M"
  elif [ "$RELEASE_TYPE" == "minor" ]; then
    reltype="m"
  elif [ "$RELEASE_TYPE" == "patch" ]; then
    reltype="p"
  else
    echo "Invalid argument: $RELEASE_TYPE"
    exit 1
  fi
  VERSION=$(./scripts/release/version.sh "-$reltype" "$PREV_VERSION")
fi

export CLI_VERSION="$VERSION"

echo "Using version $VERSION"
echo "Previous version: $PREV_VERSION"

# create and push tag
git tag -a "$VERSION" -m "$VERSION"
git push --quiet origin "$VERSION"
