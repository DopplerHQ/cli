#!/bin/bash

function finish {
  rm "$GOOGLE_APPLICATION_CREDENTIALS"
}
trap finish EXIT

echo "$GOOGLE_CREDS" > "$GOOGLE_APPLICATION_CREDENTIALS"
goreleaser release --rm-dist
scripts/publish-deb.sh
scripts/publish-rpm.sh
