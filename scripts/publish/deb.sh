#!/bin/sh

# Adapted from https://github.com/leopardslab/dunner/blob/master/release/publish_deb_to_bintray.sh

source scripts/publish/utils.sh

set -e

SUBJECT="dopplerhq"
REPO="doppler-deb"
PACKAGE="doppler"
DISTRIBUTIONS="stable"
COMPONENTS="main"

if [ -z "$BINTRAY_USER" ]; then
  echo "BINTRAY_USER is not set"
  exit 1
fi

if [ -z "$BINTRAY_API_KEY" ]; then
  echo "BINTRAY_API_KEY is not set"
  exit 1
fi

setUploadDirPath () {
  UPLOADDIRPATH="pool/s/$PACKAGE"
}

listDebianArtifacts() {
  FILES=$(find dist/*.deb  -type f)
}

cleanArtifacts () {
  rm -f "$(pwd)/*.deb"
}

cleanArtifacts
listDebianArtifacts
getVersion
printMeta
bintrayCreateVersion
bintrayUseGitHubReleaseNotes
setUploadDirPath
bintrayUpload "$UPLOADDIRPATH/$FILENAME;deb_distribution=$DISTRIBUTIONS;deb_component=$COMPONENTS;deb_architecture=$ARCH?publish=1&override=1"
snooze
bintraySetDownloads "$UPLOADDIRPATH/$FILENAME"
