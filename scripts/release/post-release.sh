#!/bin/bash

set -e

# publish binaries to bintray
scripts/publish/deb.sh
scripts/publish/rpm.sh
