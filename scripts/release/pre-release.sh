#!/bin/bash

set -e

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

GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [ "$GIT_BRANCH" != "master" ]; then
  echo "You must be on the master branch"
  exit 1
fi

if [ -z "$(command -v cloudsmith)" ]; then
  echo "cloudsmith-cli must be installed"
  exit 1
fi

echo "Using $(go version)"
read -rp "Continue? (y/n) " ok
if [ "$ok" != "y" ] && [ "$ok" != "Y" ] && [ "$ok" != "yes" ]; then
  echo "Exiting"
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

# get git in order
git push --quiet
git tag -a "$VERSION" -m "$VERSION"
git push --quiet origin "$VERSION"  # push only this tag
