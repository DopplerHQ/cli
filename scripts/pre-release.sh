#!/bin/bash

set -e

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
