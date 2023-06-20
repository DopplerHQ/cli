#!/bin/bash

set -euo pipefail

TEST_NAME="legacy-commands"

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

beforeEach() {
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
}

beforeAll

beforeEach

# test 'doppler enclave'
"$DOPPLER_BINARY" enclave > /dev/null 2>&1 || (echo "ERROR: doppler enclave" && exit 1)

beforeEach

# test 'doppler enclave configs'
"$DOPPLER_BINARY" enclave configs > /dev/null 2>&1 || (echo "ERROR: doppler enclave configs" && exit 1)

beforeEach

# test 'doppler enclave environments'
"$DOPPLER_BINARY" enclave environments > /dev/null 2>&1 || (echo "ERROR: doppler enclave environments" && exit 1)

beforeEach

# test 'doppler enclave projects'
"$DOPPLER_BINARY" enclave projects > /dev/null 2>&1 || (echo "ERROR: doppler enclave projects" && exit 1)

beforeEach

# test 'doppler enclave secrets'
"$DOPPLER_BINARY" enclave secrets > /dev/null 2>&1 || (echo "ERROR: doppler enclave secrets" && exit 1)

beforeEach

# test 'doppler enclave secrets download'
"$DOPPLER_BINARY" enclave secrets download --no-file > /dev/null 2>&1 || (echo "ERROR: doppler enclave secrets download" && exit 1)

beforeEach

# test 'doppler enclave secrets get'
"$DOPPLER_BINARY" enclave secrets get DOPPLER_CONFIG > /dev/null 2>&1 || (echo "ERROR: doppler enclave secrets get DOPPLER_CONFIG" && exit 1)

afterAll
