#!/bin/bash

function finish {
  rm -f "$GOOGLE_APPLICATION_CREDENTIALS"
}
trap finish EXIT

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

git tag -a "$TAGNAME" -m "$TAGNAME"
git push origin "$TAGNAME"

echo "$GOOGLE_CREDS" > "$GOOGLE_APPLICATION_CREDENTIALS"
goreleaser release --rm-dist
scripts/publish-deb.sh
scripts/publish-rpm.sh
