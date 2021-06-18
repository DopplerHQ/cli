#!/bin/bash

set -euo pipefail

TEST_NAME="install.sh-update-in-place"
TEMP_INSTALL_DIR="$PWD/temp-install-dir"
ORIG_DOPPLER="$(command -v doppler || true)"
PATH="$TEMP_INSTALL_DIR:$PATH"

######################################################################

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
  rm -rf "$TEMP_INSTALL_DIR"

  if test ! -z "$ORIG_DOPPLER"; then
    echo "INFO: Moving original doppler executable"
    mv "$ORIG_DOPPLER" ./doppler.orig
  fi
}

beforeEach() {
  header
  rm -rf "$TEMP_INSTALL_DIR"
}

afterEach() {
  footer
}

header() {
  echo "========================================="
  echo "EXECUTING: $name"
}

footer() {
  echo "========================================="
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  rm -rf "$TEMP_INSTALL_DIR"

  if test ! -z "$ORIG_DOPPLER"; then
    echo "INFO: Restoring original doppler executable"
    mv ./doppler.orig "$ORIG_DOPPLER"
  fi
}

md5hash() {
  md5 -rq $1 || md5sum $1 | awk '{print $1}'
}

######################################################################

beforeAll

######################################################################
#

name="custom install path works"

beforeEach

set +e
mkdir "$TEMP_INSTALL_DIR"
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --install-path $TEMP_INSTALL_DIR 2>&1)"
exit_code=$?
set -e
installed_at="$(command -v doppler || true)"

actual="$(dirname "$installed_at")"
expected="$TEMP_INSTALL_DIR"

####################

[ "$actual" == "$expected" ] || \
  (echo "ERROR: binary not installed at expected custom install path. Expected: $expected, Actual: $actual" && \
    echo "SCRIPT OUTPUT:" && echo "$output" && \
    exit 1)

afterEach

######################################################################
#

name="default /usr/local/bin works"

beforeEach

set +e
mkdir "$TEMP_INSTALL_DIR"
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --no-package-manager 2>&1)"
exit_code=$?
set -e
installed_at="$(command -v doppler || true)"

actual="$(dirname "$installed_at")"
expected="/usr/local/bin"

rm -rf /usr/local/bin/doppler

####################

[ "$actual" == "$expected" ] || \
  (echo "ERROR: binary not installed at expected default path. Expected: $expected, Actual: $actual" && \
     echo "SCRIPT OUTPUT:" && echo "$output" && \
     exit 1)

afterEach

######################################################################
#

name="updates pre-existing binary if it exists"

beforeEach

set +e
mkdir "$TEMP_INSTALL_DIR"
touch "$TEMP_INSTALL_DIR/doppler" && chmod +x "$TEMP_INSTALL_DIR/doppler"

empty_hash="$(md5hash "$TEMP_INSTALL_DIR/doppler")"
output="$("$DOPPLER_SCRIPTS_DIR/install.sh" --no-package-manager 2>&1)"
exit_code=$?
set -e

updated_hash="$(md5hash "$TEMP_INSTALL_DIR/doppler")"
installed_at="$(command -v doppler || true)"

actual_dir="$(dirname "$installed_at")"
# TEMP_INSTALL_DIR is the first dir on the PATH, so we expect that
expected_dir="$TEMP_INSTALL_DIR"

####################

[ "$expected_dir" == "$actual_dir" ] || \
  (echo "ERROR: binary not installed to expected path. Expected: $expected_dir, Actual: $actual_dir" && \
    echo "SCRIPT OUTPUT:" && echo "$output" && \
    exit 1)
[ "$updated_hash" != "$empty_hash" ] || \
  (echo "ERROR: expected binary wasn't updated" && \
    echo "SCRIPT OUTPUT:" && echo "$output" && \
    exit 1)

afterEach

######################################################################

afterAll
