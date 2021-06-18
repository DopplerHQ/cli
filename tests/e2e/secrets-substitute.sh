#!/bin/bash

set -euo pipefail

TEST_NAME="secrets-substitute"

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
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
}

beforeEach() {
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
}

error() {
  message=$1
  echo "$message"
  exit 1
}

beforeAll

beforeEach

# verify template substitution behavior
config="$("$DOPPLER_BINARY" secrets substitute /dev/stdin <<<'{{.DOPPLER_CONFIG}}')"
[[ "$config" == "e2e" ]] || error "ERROR: secrets substitute output was incorrect"

"$DOPPLER_BINARY" secrets substitute nonexistent-file.txt && \
  error "ERROR: secrets substitute did not fail on nonexistent file"

afterAll
