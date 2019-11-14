#!/bin/bash

set -e

VERSION=$(git describe --abbrev=0)

echo "Building macOS pkg"

pkgbuild --root dist/doppler_darwin_amd64/ --identifier "com.dopplerhq.cli" --version "${VERSION:1}" --install-location /usr/local/bin doppler.pkg
productbuild --package doppler.pkg "doppler-$VERSION.pkg"
