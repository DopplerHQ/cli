#!/bin/sh

# Adapted from https://github.com/leopardslab/dunner/blob/master/release/publish_rpm_to_bintray.sh

source scripts/publish/utils.sh

set -e

SUBJECT="dopplerhq"
REPO="doppler-rpm"
PACKAGE="doppler"

if [ -z "$BINTRAY_USER" ]; then
  echo "BINTRAY_USER is not set"
  exit 1
fi

if [ -z "$BINTRAY_API_KEY" ]; then
  echo "BINTRAY_API_KEY is not set"
  exit 1
fi

listRpmArtifacts() {
  FILES=$(find dist/*.rpm  -type f)
}

cleanArtifacts () {

  rm -f "$(pwd)/*.rpm"
}

cleanArtifacts
listRpmArtifacts
getVersion
printMeta
bintrayCreateVersion
bintrayUseGitHubReleaseNotes
bintrayUpload "$ARCH/$FILENAME?publish=1&override=1"
snooze
bintraySetDownloads "$ARCH/$FILENAME"
