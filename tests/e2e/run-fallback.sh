#!/bin/bash

set -euo pipefail

TEST_NAME="run-fallback"

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
  rm -f fallback.json nonexistent-fallback.json
  rm -rf ./temp-fallback
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
  rm -f fallback.json nonexistent-fallback.json
  rm -rf ./temp-fallback
}

beforeAll

beforeEach

# test fallback-only fails when no fallback files exist
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null 2>&1 && (echo "ERROR: --fallback-only flag is not respected" && exit 1)

beforeEach

# test fallback-readonly doesn't write a fallback file
"$DOPPLER_BINARY" run --fallback-readonly -- echo -n > /dev/null
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null 2>&1 && (echo "ERROR: --fallback-readonly flag is not respected" && exit 1)

beforeEach

# test 'run' respects custom fallback file location
"$DOPPLER_BINARY" run --fallback ./fallback.json -- echo -n > /dev/null
# should fail due to non-existence of default fallback file
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null 2>&1 && (echo "ERROR: --fallback flag is not respected" && exit 1)
"$DOPPLER_BINARY" run --fallback ./fallback.json --fallback-only -- echo -n > /dev/null 2>&1 || (echo "ERROR: --fallback flag is not respected" && exit 1)
rm -f fallback.json

beforeEach

# test 'run' respects custom passphrase
"$DOPPLER_BINARY" run --passphrase=123456 -- echo -n > /dev/null
# ensure default passphrase fails
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null 2>&1 && (echo "ERROR: --passphrase flag is not respected" && exit 1)
# test decryption with custom passphrase
"$DOPPLER_BINARY" run --fallback-only --passphrase=123456 -- echo -n > /dev/null || (echo "ERROR: --passphrase2 flag is not respected" && exit 1)

beforeEach

# test 'run' respects custom passphrase from environment
DOPPLER_PASSPHRASE=123456 "$DOPPLER_BINARY" run -- echo -n > /dev/null
# ensure default passphrase fails
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null 2>&1 && (echo "ERROR: --passphrase flag is not respected (1)" && exit 1)
# test decryption with custom passphrase flag
"$DOPPLER_BINARY" run --fallback-only --passphrase=123456 -- echo -n > /dev/null || (echo "ERROR: --passphrase flag is not respected (2)" && exit 1)
# test decryption with custom passphrase from environment
DOPPLER_PASSPHRASE=123456 "$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null || (echo "ERROR: --passphrase flag is not respected (3)" && exit 1)

beforeEach

# test 'run' respects --no-exit-on-write-failure
mkdir ./temp-fallback
chmod 500 ./temp-fallback
# this should fail
"$DOPPLER_BINARY" run --fallback=./temp-fallback -- echo -n > /dev/null 2>&1 && (echo "ERROR: --no-exit-on-write-failure flag is not respected" && exit 1)
# this should succeed
"$DOPPLER_BINARY" run --fallback=./temp-fallback --no-exit-on-write-failure -- echo -n > /dev/null || (echo "ERROR: --no-exit-on-write-failure flag is not respected" && exit 1)
rm -rf ./temp-fallback

beforeEach

# test 'run' w/ no cache and invalid fallback file
"$DOPPLER_BINARY" run --fallback ./fallback.json -- echo -n > /dev/null
rm -f fallback.json

beforeEach

# test 'run' w/ valid cache and invalid fallback file
"$DOPPLER_BINARY" run --fallback ./fallback.json -- echo -n > /dev/null
rm -f fallback.json
echo "foo" > ./fallback.json
"$DOPPLER_BINARY" run --fallback ./fallback.json -- echo -n > /dev/null || (echo "ERROR: run w/ valid cache is not ignoring invalid fallback file" && exit 1)
rm -f fallback.json

beforeEach

# test 'run' w/ valid cache and non-existent fallback file
"$DOPPLER_BINARY" run --fallback ./fallback.json -- echo -n > /dev/null
"$DOPPLER_BINARY" run --fallback ./nonexistent-fallback.json -- echo -n > /dev/null || (echo "ERROR: run w/ valid cache is not ignoring nonexistent fallback file" && exit 1)
rm -f fallback.json nonexistent-fallback.json

afterAll
