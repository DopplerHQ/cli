#!/bin/bash

set -euo pipefail

DIR="$(dirname "$0")"

export DOPPLER_BINARY="$DIR/doppler"
export DOPPLER_PROJECT="cli"
export DOPPLER_CONFIG="prd_e2e_tests"

# Run tests
"$DIR/tests/secrets-download-fallback.sh"

echo -e "\nAll tests completed successfully!"
exit 0
