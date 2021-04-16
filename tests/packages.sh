#!/bin/bash

set -e

# deb
echo "Testing ubuntu"
time docker run --rm -it -v "$(pwd)/tests/packages":/usr/doppler/cli/packages:ro ubuntu /usr/doppler/cli/packages/deb.sh
echo "Testing debian"
time docker run --rm -it -v "$(pwd)/tests/packages":/usr/doppler/cli/packages:ro debian /usr/doppler/cli/packages/deb.sh

# rpm
echo "Testing centos"
time docker run --rm -it -v "$(pwd)/tests/packages":/usr/doppler/cli/packages:ro centos /usr/doppler/cli/packages/rpm.sh
echo "Testing fedora"
time docker run --rm -it -v "$(pwd)/tests/packages":/usr/doppler/cli/packages:ro fedora /usr/doppler/cli/packages/rpm.sh

# apk
echo "Testing alpine"
time docker run --rm -it -v "$(pwd)/tests/packages":/usr/doppler/cli/packages:ro alpine /usr/doppler/cli/packages/apk.sh
