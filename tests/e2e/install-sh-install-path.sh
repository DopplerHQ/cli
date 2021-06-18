#!/bin/bash

set -euo pipefail

TEST_NAME="install.sh-install-path"

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
  rm -rf ./temp-install-dir
}

beforeEach() {
  rm -rf ./temp-install-dir
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  rm -rf ./temp-install-dir
}

beforeAll

beforeEach

# valid install path
mkdir ./temp-install-dir
"$DOPPLER_SCRIPTS_DIR/install.sh" --install-path ./temp-install-dir > /dev/null 2>&1
[ -e ./temp-install-dir/doppler ] || (echo "ERROR: --install-path flag is not respected" && exit 1)

beforeEach

# install path doesn't exist
set +e
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --install-path ./temp-install-dir 2>&1)"
exit_code=$?
set -e
[ "$exit_code" -eq 1 ] || \
  (echo "ERROR: incorrect exit code when install path doesn't exist" && exit 1)
[ "$(echo "$output" | tail -1)" == "Install path does not exist: \"./temp-install-dir\"" ] || \
  (echo "ERROR: incorrect error message when install path doesn't exist" && exit 1)

beforeEach

# install path exists but isn't a directory
touch ./temp-install-dir
set +e
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --install-path ./temp-install-dir 2>&1)"
exit_code=$?
set -e
[ "$exit_code" -eq 1 ] || \
  (echo "ERROR: incorrect exit code when install path isn't a directory" && exit 1)
[ "$(echo "$output" | tail -1)" == "Install path is not a valid directory: \"./temp-install-dir\"" ] || \
  (echo "ERROR: incorrect error message when install path isn't a directory" && exit 1)

beforeEach

# install path is a directory lacking write perms
mkdir -m 500 ./temp-install-dir
set +e
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --install-path ./temp-install-dir 2>&1)"
exit_code=$?
set -e
[ "$exit_code" -eq 2 ] || \
  (echo "ERROR: incorrect exit code when install path lacks write perms" && exit 1)
[ "$(echo "$output" | tail -1)" == "Install path is not writable: \"./temp-install-dir\"" ] || \
  (echo "ERROR: incorrect error message when install path lacks write perms" && exit 1)

afterAll
