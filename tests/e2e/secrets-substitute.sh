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

export MY_ENV_VAR="123"
export TEST="foo"

# DOPPLER_ENVIRONMENT is used here because it isn't specified as an environment variable for the purposes of configuration

# verify default template substitution behavior
config="$("$DOPPLER_BINARY" secrets substitute /dev/stdin <<<'{{.DOPPLER_ENVIRONMENT}}')"
[[ "$config" == "e2e" ]] || error "ERROR: secrets substitute output was incorrect"

"$DOPPLER_BINARY" secrets substitute nonexistent-file.txt &&
  error "ERROR: secrets substitute did not fail on nonexistent file"

output="$("$DOPPLER_BINARY" secrets substitute /dev/stdin --use-env false <<<'{{.DOPPLER_ENVIRONMENT}} {{.MY_ENV_VAR}} {{.TEST}}')"
[[ "$output" == "e2e <no value> abc" ]] || error "ERROR: secrets substitute output was incorrect (env:false)"

output="$("$DOPPLER_BINARY" secrets substitute /dev/stdin --use-env true <<<'{{.DOPPLER_ENVIRONMENT}} {{.MY_ENV_VAR}} {{.TEST}}')"
[[ "$output" == "e2e 123 abc" ]] || error "ERROR: secrets substitute output was incorrect (env:true)"

output="$("$DOPPLER_BINARY" secrets substitute /dev/stdin --use-env override <<<'{{.DOPPLER_ENVIRONMENT}} {{.MY_ENV_VAR}} {{.TEST}}')"
[[ "$output" == "e2e 123 foo" ]] || error "ERROR: secrets substitute output was incorrect (env:override)"

output="$("$DOPPLER_BINARY" secrets substitute /dev/stdin --use-env only <<<'{{.DOPPLER_ENVIRONMENT}} {{.MY_ENV_VAR}} {{.TEST}}')"
[[ "$output" == "<no value> 123 foo" ]] || error "ERROR: secrets substitute output was incorrect (env:only)"

output="$(DOPPLER_TOKEN="invalid" "$DOPPLER_BINARY" secrets substitute /dev/stdin --use-env only <<<'{{.DOPPLER_ENVIRONMENT}} {{.MY_ENV_VAR}} {{.TEST}}')"
[[ "$output" == "<no value> 123 foo" ]] || error "ERROR: secrets substitute output was incorrect (env:only token:cleared)"

afterAll
