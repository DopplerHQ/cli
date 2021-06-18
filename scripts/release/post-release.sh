#!/bin/bash

set -euo pipefail

# print currently configured user to aid with debugging
cloudsmith whoami

publishToCloudsmith() {
  TYPE="$1"
  DISTRO="$2"
  VERSION="$3"
  PACKAGES=$4
  for i in $PACKAGES; do
    PACKAGE=${i##*/}
    # we can't upload both armv6 and armv7, so use armv7
    if [[ "$PACKAGE" == *"armv6"* ]]; then
      echo "Skipping $PACKAGE"
      continue
    fi

    echo "Uploading $PACKAGE"
    # attempt each publish up to 3 times
    cloudsmith push "$TYPE" "doppler/cli/$DISTRO/$VERSION" "dist/$PACKAGE" || \
      cloudsmith push "$TYPE" "doppler/cli/$DISTRO/$VERSION" "dist/$PACKAGE" || \
      cloudsmith push "$TYPE" "doppler/cli/$DISTRO/$VERSION" "dist/$PACKAGE"
  done;
}

# publish deb packages to cloudsmith
PACKAGES=$(find dist/*.deb  -type f)
publishToCloudsmith deb any-distro any-version "$PACKAGES"

# publish rpm packages to cloudsmith
PACKAGES=$(find dist/*.rpm  -type f)
publishToCloudsmith rpm any-distro any-version "$PACKAGES"

# publish alpine packages to cloudsmith
PACKAGES=$(find dist/*.apk  -type f)
publishToCloudsmith alpine alpine any-version "$PACKAGES"

# send Slack notification
CHANGELOG="$(doppler changelog -n 1 --no-check-version | tail -n +2)"
# escape characters for slack https://api.slack.com/reference/surfaces/formatting#escaping
CHANGELOG=${CHANGELOG//&/&amp;}
CHANGELOG=${CHANGELOG//</&lt;}
CHANGELOG=${CHANGELOG//>/&gt;}
# escape double quotes
CHANGELOG=${CHANGELOG//\"/\\\"}
# replace newlines with newline character
CHANGELOG=${CHANGELOG/$'\n'/'\\n'}

VERSION=$(git describe --abbrev=0)
MESSAGE="Doppler CLI <https://github.com/DopplerHQ/cli/releases/tag/$VERSION|v$VERSION> has been released. Changelog:\n$CHANGELOG"
curl --tlsv1.2 --proto "=https" -s -X "POST" "$SLACK_WEBHOOK_URL" -H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
  --data-urlencode "payload={\"username\": \"CLI Release Bot\", \"text\": \"$MESSAGE\"}"
