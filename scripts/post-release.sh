#!/bin/bash

set -e

VERSION=$(git describe --abbrev=0)
# remove leading 'v'
VERSION=${VERSION:1}

echo "Building macOS pkg"

pkg_name="doppler.pkg"
final_pkg_name="doppler_$VERSION.pkg"

# we must use the GitHub release id
github_release_id=$(curl --silent --show-error https://api.github.com/repos/DopplerHQ/cli/releases/latest  | jq '.id')

# build the package
pkgbuild --root dist/doppler_darwin_amd64/ --identifier "com.dopplerhq.cli" --version "$VERSION" --install-location /usr/local/bin "dist/$pkg_name"
productbuild --package "dist/$pkg_name" "dist/$final_pkg_name"
rm "dist/$pkg_name"

# upload the package
echo "Uploading macOS pkg to GitHub"
URL="https://uploads.github.com/repos/DopplerHQ/cli/releases/$github_release_id/assets?name=$final_pkg_name"
RESPONSE_CODE=$(curl -T "dist/$final_pkg_name" -X POST -u "$GITHUB_USER:$GITHUB_TOKEN" -H "Content-Type: application/octet-stream" $URL -s -w "%{http_code}" -o /dev/null)
if [[ "$(echo $RESPONSE_CODE | head -c2)" != "20" ]]; then
  echo "Unable to upload, HTTP response code: $RESPONSE_CODE"
  exit 1
fi

# publish binaries to bintray
scripts/publish-deb.sh
scripts/publish-rpm.sh
