#!/bin/bash

set -euo pipefail

DIR="$(dirname "$0")"

export DOPPLER_BINARY="$DIR/../doppler"
export DOPPLER_SCRIPTS_DIR="$DIR/../scripts"
export DOPPLER_PROJECT="cli"
export DOPPLER_CONFIG="prd_e2e_tests"

# Run tests
"$DIR/e2e/secrets-download-fallback.sh"
"$DIR/e2e/run-fallback.sh"
"$DIR/e2e/configure.sh"
"$DIR/e2e/install-sh-install-path.sh"

echo -e "\nAll tests completed successfully!"
exit 0
