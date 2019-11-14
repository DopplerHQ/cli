#!/bin/bash

set -e

VERSION=$(git describe --abbrev=0)

echo "Building macOS pkg"

pkg_name="dist/doppler.pkg"

pkgbuild --root dist/doppler_darwin_amd64/ --identifier "com.dopplerhq.cli" --version "${VERSION:1}" --install-location /usr/local/bin "$pkg_name"
productbuild --package "$pkg_name" "dist/doppler-$VERSION.pkg"
rm "$pkg_name"
