#!/bin/bash

set -e

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

RELEASE_TYPE=$1
PREV_VERSION=$(git describe --abbrev=0)

if [ "${RELEASE_TYPE:0:1}" == "v" ]; then
  VERSION="$RELEASE_TYPE"
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
  VERSION=$(./scripts/version.sh "-$reltype" "$PREV_VERSION")
fi

echo "Using version $VERSION"
echo "Previous version: $PREV_VERSION"

# get git in order
git push --quiet
git tag -a "$VERSION" -m "$VERSION"
git push origin "$VERSION"  # push only this tag
