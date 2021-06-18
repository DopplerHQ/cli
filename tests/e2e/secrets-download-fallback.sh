#!/bin/bash

set -euo pipefail

TEST_NAME="secrets-download-fallback"

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
  rm -f fallback.json secrets.yaml doppler.env
  rm -rf ./temp-fallback
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
  rm -f fallback.json secrets.yaml doppler.env
  rm -rf ./temp-fallback
}

beforeAll

beforeEach

# test fallback-only fails when no fallback files exist
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: --fallback-only flag is not respected" && exit 1)

beforeEach

# test fallback-readonly doesn't write a fallback file
"$DOPPLER_BINARY" secrets download --no-file --fallback-readonly > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: --fallback-readonly flag is not respected" && exit 1)

beforeEach

# test 'secrets download' respects custom fallback file location
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null
# should fail due to non-existence of default fallback file
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: --fallback flag is not respected" && exit 1)
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json --fallback-only > /dev/null 2>&1 || (echo "ERROR: --fallback flag is not respected" && exit 1)
rm -f fallback.json

beforeEach

# test 'secrets download' respects custom passphrase
"$DOPPLER_BINARY" secrets download --no-file --fallback-passphrase=123456 > /dev/null
# ensure default passphrase fails
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: --passphrase flag is not respected" && exit 1)
# test decryption with custom passphrase
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback-passphrase=123456 > /dev/null || (echo "ERROR: --passphrase flag is not respected" && exit 1)

beforeEach

# test 'secrets download' respects custom passphrase from environment
DOPPLER_PASSPHRASE=123456 "$DOPPLER_BINARY" secrets download --no-file > /dev/null
# ensure default passphrase fails
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: --passphrase flag is not respected (1)" && exit 1)
# test decryption with custom passphrase flag
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback-passphrase=123456 > /dev/null || (echo "ERROR: --passphrase flag is not respected (2)" && exit 1)
# test decryption with custom passphrase from environment
DOPPLER_PASSPHRASE=123456 "$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null || (echo "ERROR: --passphrase flag is not respected (3)" && exit 1)

beforeEach

# test 'secrets download' respects --no-exit-on-write-failure
mkdir ./temp-fallback
chmod 500 ./temp-fallback
# this should fail
"$DOPPLER_BINARY" secrets download --no-file --fallback=./temp-fallback > /dev/null 2>&1 && (echo "ERROR: --no-exit-on-write-failure flag is not respected" && exit 1)
# this should succeed
"$DOPPLER_BINARY" secrets download --no-file --fallback=./temp-fallback --no-exit-on-write-failure > /dev/null || (echo "ERROR: --no-exit-on-write-failure flag is not respected" && exit 1)
rm -rf ./temp-fallback

beforeEach

# test fallback file contents matches api response
a="$("$DOPPLER_BINARY" secrets download --no-file)"
b="$("$DOPPLER_BINARY" secrets download --no-file --fallback-only)"
[[ "$a" == "$b" ]] || (echo "ERROR: fallback file contents do not match" && exit 1)

beforeEach

# test 'run' fallback file contents matches 'secrets download' api response
# generate fallback file
"$DOPPLER_BINARY" run -- echo -n
# print fallback file
a="$("$DOPPLER_BINARY" secrets download --no-file --fallback-only)"
# fetch from api
b="$("$DOPPLER_BINARY" secrets download --no-file --no-fallback)"
[[ "$a" == "$b" ]] || (echo "ERROR: 'run' fallback file contents do not match 'secrets download'" && exit 1)

beforeEach

# test 'secrets download' fallback file can be used with 'run'
# generate fallback file
"$DOPPLER_BINARY" secrets download --no-file > /dev/null
"$DOPPLER_BINARY" run --fallback-only -- echo -n > /dev/null || (echo "ERROR: 'secrets download' fallback file is not used by 'run'" && exit 1)

beforeEach

# test 'secrets download' file can be used as fallback file for 'run'
"$DOPPLER_BINARY" secrets download --no-fallback ./fallback.json  > /dev/null
"$DOPPLER_BINARY" run --fallback ./fallback.json --fallback-only > /dev/null -- echo -n || (echo "ERROR: 'secrets download' file failed to be used as fallback file for 'run'" && exit 1)
rm -f fallback.json

beforeEach

# test 'secrets download' file can be used as fallback file for 'secrets download'
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback ./fallback.json > /dev/null || (echo "ERROR: 'secrets download' file failed to be used as fallback file for 'secrets download'" && exit 1)
rm -f fallback.json

beforeEach

# test contents of 'secrets download' file when used as fallback file for 'secrets download'
a="$("$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json)"
b="$("$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback ./fallback.json)"
rm -f fallback.json
[[ "$a" == "$b" ]] || (echo "ERROR: 'secrets download' file contents do not match when used as fallback file for 'secrets download'" && exit 1)

beforeEach

# test 'secrets download' writes correct file name when format is env
"$DOPPLER_BINARY" secrets download --format=env > /dev/null
[[ -f doppler.env ]] || (echo "ERROR: 'secrets download' did not save doppler.env when format is env" && exit 1)
rm -f ./doppler.env

beforeEach

# test 'secrets download' doesn't write fallback when format is env
"$DOPPLER_BINARY" secrets download --no-file --format=env > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: 'secrets download' should not write fallback file when format is env" && exit 1)

beforeEach

# test 'secrets download' ignores fallback flags when format is env
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback=./nonexistent-file --format=env > /dev/null

beforeEach

# test 'secrets download' writes correct file name when format is yaml
"$DOPPLER_BINARY" secrets download --format=yaml > /dev/null
[[ -f secrets.yaml ]] || (echo "ERROR: 'secrets download' did not save secrets.yaml when format is yaml" && exit 1)
rm -f ./secrets.yaml

beforeEach

# test 'secrets download' doesn't write fallback when format is yaml
"$DOPPLER_BINARY" secrets download --no-file --format=yaml > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: 'secrets download' should not write fallback file when format is yaml" && exit 1)

beforeEach

# test 'secrets download' ignores fallback flags when format is yaml
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback=./nonexistent-file --format=yaml > /dev/null

beforeEach

# test 'secrets download' doesn't write fallback when format is docker
"$DOPPLER_BINARY" secrets download --no-file --format=docker > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback-only > /dev/null 2>&1 && (echo "ERROR: 'secrets download' should not write fallback file when format is docker" && exit 1)

beforeEach

# test 'secrets download' ignores fallback flags when format is docker
"$DOPPLER_BINARY" secrets download --no-file --fallback-only --fallback=./nonexistent-file --format=docker > /dev/null

beforeEach

# test 'secrets download' w/ no cache and invalid fallback file
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null
rm -f fallback.json

beforeEach

# test 'secrets download' w/ valid cache and invalid fallback file
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null
rm -f fallback.json
echo "foo" > ./fallback.json
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null || (echo "ERROR: run w/ valid cache is not ignoring invalid fallback file" && exit 1)
rm -f fallback.json

beforeEach

# test 'secrets download' w/ valid cache and non-existent fallback file
"$DOPPLER_BINARY" secrets download --no-file --fallback ./fallback.json > /dev/null
"$DOPPLER_BINARY" secrets download --no-file --fallback ./nonexistent-fallback.json > /dev/null || (echo "ERROR: run w/ valid cache is not ignoring nonexistent fallback file" && exit 1)
rm -f fallback.json

afterAll
