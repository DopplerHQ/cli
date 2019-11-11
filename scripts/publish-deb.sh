#!/bin/sh

# Adapted from https://github.com/leopardslab/dunner/blob/master/release/publish_deb_to_bintray.sh

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

getVersion () {
  VERSION=$(git describe);
  if [ "${VERSION:0:1}" == "v" ]
  then
    VERSION="${VERSION:1}"
  fi
}

setUploadDirPath () {
  UPLOADDIRPATH="pool/s/$PACKAGE"
}

listDebianArtifacts() {
  FILES=$(find dist/*.deb  -type f)
}

bintrayUpload () {
  for i in $FILES; do
    FILENAME=${i##*/}
    ARCH=$(echo ${FILENAME##*_} | cut -d '.' -f 1)
    if [ $ARCH == "64" ]; then
      ARCH="x86_64"
    fi

    URL="https://api.bintray.com/content/$SUBJECT/$REPO/$PACKAGE/$VERSION/$UPLOADDIRPATH/$FILENAME;deb_distribution=$DISTRIBUTIONS;deb_component=$COMPONENTS;deb_architecture=$ARCH?publish=1&override=1"
    echo "Uploading $URL"

    RESPONSE_CODE=$(curl -T $i -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -I -s -w "%{http_code}" -o /dev/null);
    if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
      echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
      exit 1
    fi
    echo "HTTP response code: $RESPONSE_CODE"
  done;
}

bintraySetDownloads () {
  for i in $FILES; do
    FILENAME=${i##*/}
    ARCH=$(echo ${FILENAME##*_} | cut -d '.' -f 1)
    if [ $ARCH == "64" ]; then
      ARCH="x86_64"
    fi
    URL="https://api.bintray.com/file_metadata/$SUBJECT/$REPO/$UPLOADDIRPATH/$FILENAME"

    echo "Putting $FILENAME in $PACKAGE's download list"
    RESPONSE_CODE=$(curl -X PUT -d "{ \"list_in_downloads\": true }" -H "Content-Type: application/json" -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -s -w "%{http_code}" -o /dev/null);

    if [ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]; then
        echo "Unable to put in download list, HTTP response code: $RESPONSE_CODE"
        exit 1
    fi
    echo "HTTP response code: $RESPONSE_CODE"
  done
}

snooze () {
    echo "\nSleeping for 30 seconds. Have a coffee..."
    sleep 30s;
    echo "Done sleeping\n"
}

printMeta () {
    echo "Publishing: $PACKAGE"
    echo "Version to be uploaded: $VERSION"
}

cleanArtifacts () {
  rm -f "$(pwd)/*.deb"
}

cleanArtifacts
listDebianArtifacts
getVersion
printMeta
setUploadDirPath
bintrayUpload
snooze
bintraySetDownloads
