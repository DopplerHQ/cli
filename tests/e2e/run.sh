#!/bin/bash

set -euo pipefail

TEST_NAME="run"

cleanup() {
  exit_code=$?
  if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: '$TEST_NAME' tests failed during execution"
    afterAll || echo "ERROR: Cleanup failed"
  fi

  exit "$exit_code"
}
trap cleanup EXIT

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

# verify local env is ignored
config="$(DOPPLER_CONFIG=123 "$DOPPLER_BINARY" run --config e2e -- printenv DOPPLER_CONFIG)"
[[ "$config" == "e2e" ]] || error "ERROR: conflicting local env var is not ignored"

beforeEach

# verify local env is used when specifying --preserve-env
config="$(DOPPLER_CONFIG=123 "$DOPPLER_BINARY" run --config e2e --preserve-env -- printenv DOPPLER_CONFIG)"
[[ "$config" == "123" ]] || error "ERROR: conflicting local env var is not used with --preserve-env"

beforeEach

# verify local env is used when key isn't specified in Doppler
value="$(NONEXISTENT_KEY=123 "$DOPPLER_BINARY" run -- printenv NONEXISTENT_KEY)"
[[ "$value" == "123" ]] || error "ERROR: local env var is not used"

beforeEach

# verify local env is used when key isn't specified in Doppler (and --preserve-env is specified)
value="$(NONEXISTENT_KEY=123 "$DOPPLER_BINARY" run --preserve-env -- printenv NONEXISTENT_KEY)"
[[ "$value" == "123" ]] || error "ERROR: local env var is not used with --preserve-env"

beforeEach

# verify reserved secrets are ignored
# first verify the doppler config has a secret named 'HOME', or this test is useless
"$DOPPLER_BINARY" secrets get HOME >/dev/null 2>&1 || error "ERROR: doppler config does not contain 'HOME' secret"
home="$HOME"
value="$("$DOPPLER_BINARY" run -- printenv HOME)"
[[ "$value" == "$home" ]] || error "ERROR: reserved secret is not ignored"

beforeEach

# verify command's exit code is properly returned
exit_code=0
"$DOPPLER_BINARY" run --command "true && exit 7" || exit_code=$?
[[ $exit_code == 7 ]] || error "ERROR: invalid exit code"

beforeEach

# verify proper quote handling
value="$("$DOPPLER_BINARY" run echo "a'b" "c'd")"
[[ "$value" == "a'b c'd" ]] || error "ERROR: quotes are improperly handled"

beforeEach

# verify flags specified after '--' are passed to subcommand
"$DOPPLER_BINARY" run -- true --config invalidconfig || error "ERROR: flags specified after '--' are improperly handled"

afterAll
