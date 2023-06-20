#!/bin/bash

set -euo pipefail

TEST_NAME="me"

cleanup() {
  exit_code=$?
  if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: '$TEST_NAME' tests failed during execution"
    afterAll || echo "ERROR: Cleanup failed"
  fi

  exit "$exit_code"
}
trap cleanup EXIT
trap cleanup INT

beforeAll() {
  echo "INFO: Executing '$TEST_NAME' tests"
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
}

error() {
  message=$1
  echo "$message"
  exit 1
}

beforeAll

# verify valid token exit code
"$DOPPLER_BINARY" me >/dev/null || error "ERROR: valid token produced error"

# verify invalid token exit code
DOPPLER_TOKEN="invalid" "$DOPPLER_BINARY" me >/dev/null && error "ERROR: invalid token did not produce error"

afterAll
