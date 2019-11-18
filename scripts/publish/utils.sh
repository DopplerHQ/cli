printMeta () {
  echo "Publishing: $PACKAGE"
  echo "Version to be uploaded: $VERSION"
}

snooze () {
  echo "\nSleeping for 30 seconds. Have a coffee..."
  sleep 30s;
  echo "Done sleeping\n"
}

getVersion () {
  # use --abbrev=0 to ensure the returned tag was explicitly set
  VERSION=$(git describe --abbrev=0);
}

bintrayCreateVersion () {
  URL="https://api.bintray.com/packages/$SUBJECT/$REPO/$PACKAGE/versions"
  BODY="{ \"name\": \"$VERSION\", \"github_use_tag_release_notes\": false, \"vcs_tag\": \"$VERSION\" }"
  echo "Creating package version $VERSION"
  RESPONSE_CODE=$(curl -X POST -d "$BODY" -H "Content-Type: application/json" -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -s -w "%{http_code}" -o /dev/null);
  if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
    echo "Unable to create package version, HTTP response code: $RESPONSE_CODE"
    exit 1
  fi
}

bintrayUseGitHubReleaseNotes () {
  URL="https://api.bintray.com/packages/$SUBJECT/$REPO/$PACKAGE/versions/$VERSION"
  BODY="{ \"vcs_tag\": \"v$VERSION\", \"github_use_tag_release_notes\": \"true\" }"
  echo "Using release notes from GitHub"
  RESPONSE_CODE=$(curl -X PATCH -d "$BODY" -H "Content-Type: application/json" -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -s -w "%{http_code}" -o /dev/null);
  if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
    echo "Unable to create package version, HTTP response code: $RESPONSE_CODE"
    exit 1
  fi
}

bintraySetDownloads () {
  uri=$1

  for i in $FILES; do
    FILENAME=${i##*/}
    ARCH=$(echo ${FILENAME##*_} | cut -d '.' -f 1)
    URL="https://api.bintray.com/file_metadata/$SUBJECT/$REPO/$uri"

    echo "Putting $FILENAME in $PACKAGE's download list"
    RESPONSE_CODE=$(curl -X PUT -d "{ \"list_in_downloads\": true }" -H "Content-Type: application/json" -u$BINTRAY_USER:$BINTRAY_API_KEY $URL -s -w "%{http_code}" -o /dev/null);

    if [ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]; then
        echo "Unable to put in download list, HTTP response code: $RESPONSE_CODE"
        exit 1
    fi
  done
}
