#!/bin/bash

set -euo pipefail

DIR="$(dirname "$0")"

export DOPPLER_BINARY="$DIR/../doppler"
export DOPPLER_SCRIPTS_DIR="$DIR/../scripts"
export DOPPLER_PROJECT="cli"
export DOPPLER_CONFIG="e2e"

# Run tests
"$DIR/e2e/secrets-download-fallback.sh"
"$DIR/e2e/secrets-substitute.sh"
"$DIR/e2e/run.sh"
"$DIR/e2e/run-fallback.sh"
"$DIR/e2e/run-mount.sh"
"$DIR/e2e/configure.sh"
"$DIR/e2e/install-sh-install-path.sh"
"$DIR/e2e/install-sh-update-in-place.sh"
"$DIR/e2e/legacy-commands.sh"
"$DIR/e2e/analytics.sh"
"$DIR/e2e/setup.sh"
"$DIR/e2e/me.sh"
"$DIR/e2e/global-flags.sh"
"$DIR/e2e/update.sh"

echo -e "\nAll tests completed successfully!"
exit 0
