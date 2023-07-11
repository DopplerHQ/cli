#!/bin/bash

set -euo pipefail

TEST_NAME="update"

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
  rm -rf ./temp-config
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  beforeEach
}

error() {
  message=$1
  echo "$message"
  exit 1
}

beforeAll

beforeEach

### update fails w/o sudo
output="$("$DOPPLER_BINARY" update --force 2>&1 || true)";
[ "$(echo "$output" | tail -1)" == "Doppler Error: exit status 2" ] || error "ERROR: expected update to fail without sudo"

beforeEach

### gnupg perms issue
# make gnupg directory inaccessible
sudo chown root ~/.gnupg;
output="$("$DOPPLER_BINARY" update --force 2>&1 || true)";
[ "$(echo "$output" | tail -1)" == "Doppler Error: exit status 4" ] || error "ERROR: expected update to fail without access to gnupg"
# restore gnupg directory perms
sudo chown "$(id -un)" ~/.gnupg;

beforeEach

### successful update
sudo "$DOPPLER_BINARY" update --force >/dev/null 2>&1 || error "ERROR: unable to update CLI"


afterAll
