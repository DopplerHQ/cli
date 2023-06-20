#!/bin/bash

set -euo pipefail

TEST_NAME="global flags"

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
  rm -f ./temp-config
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

###
# --no-read-env flag
###

beforeEach

# verify env var is read
token="$(DOPPLER_TOKEN=123 "$DOPPLER_BINARY" configure debug --json --configuration=./temp-config 2>/dev/null | jq -r ".[\"/\"].token")"
[[ "$token" == "123" ]] || error "ERROR: expected token from environment"
# verify env var is ignored
token="$(DOPPLER_TOKEN=123 "$DOPPLER_BINARY" configure debug --json --configuration=./temp-config --no-read-env 2>/dev/null | jq -r ".[\"/\"].token")"
[[ "$token" == "" ]] || error "ERROR: expected blank config value"

###
# configuration hierarchy
###

CONFIG_VALUE="123"
ENV_VALUE="456"
FLAG_VALUE="789"

beforeEach

# verify config value used when no env value or flag
"$DOPPLER_BINARY" configure set token "$CONFIG_VALUE" --scope=/ --configuration=./temp-config >/dev/null 2>&1
token="$("$DOPPLER_BINARY" configure debug --json --configuration=./temp-config --no-read-env 2>/dev/null | jq -r ".[\"/\"].token")"
[[ "$token" == "$CONFIG_VALUE" ]] || error "ERROR: expected token from config file"

beforeEach

# verify env value used over config value
"$DOPPLER_BINARY" configure set token "$CONFIG_VALUE" --scope=/ --configuration=./temp-config >/dev/null 2>&1
token="$(DOPPLER_TOKEN="$ENV_VALUE" "$DOPPLER_BINARY" configure debug --json --configuration=./temp-config 2>/dev/null | jq -r ".[\"/\"].token")"
[[ "$token" == "$ENV_VALUE" ]] || error "ERROR: expected token from environment"

beforeEach

# verify flag value used over env value and config value
"$DOPPLER_BINARY" configure set token "$CONFIG_VALUE" --scope=/ --configuration=./temp-config >/dev/null 2>&1
token="$(DOPPLER_TOKEN="$ENV_VALUE" "$DOPPLER_BINARY" configure debug --json --token="$FLAG_VALUE" --configuration=./temp-config 2>/dev/null | jq -r ".[\"/\"].token")"
[[ "$token" == "$FLAG_VALUE" ]] || error "ERROR: expected token from flag"

afterAll
