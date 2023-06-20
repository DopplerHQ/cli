#!/bin/bash

set -euo pipefail

TEST_NAME="config file"

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
  rm -rf ./temp-config ./temp-config-dir
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

# test get blank config
config="$("$DOPPLER_BINARY" configure --configuration=./temp-config --scope=/ --json)"
[[ "$config" == "{}" ]] || error "ERROR: unexpectd blank config value"

beforeEach

# test unset blank config
"$DOPPLER_BINARY" configure unset config --configuration=./temp-config --scope=/ --silent || error "ERROR: 'configure' failed to unset blank config"

beforeEach

# test configure
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"/":{"enclave.config":"123"}}' ]] || error "ERROR: config contents do not match"

beforeEach

# test get value
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: config contents do not match"

beforeEach

# test get value w/ --plain
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --plain)"
[[ "$config" == "123" ]] || error "ERROR: config --plain contents do not match"

beforeEach

# test configure w/ custom config-dir
mkdir ./temp-config-dir
"$DOPPLER_BINARY" configure set config 123 --config-dir=./temp-config-dir --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config-dir/.doppler.yaml --scope=/ --plain)"
[[ "$config" == "123" ]] || error "ERROR: config-dir not properly used"

beforeEach

# test configure w/ custom config-dir from environment
mkdir ./temp-config-dir
DOPPLER_CONFIG_DIR=./temp-config-dir "$DOPPLER_BINARY" configure set config 123 --scope=/ --silent
config="$(DOPPLER_CONFIG_DIR=./temp-config-dir "$DOPPLER_BINARY" configure get config --scope=/ --plain)"
[[ "$config" == "123" ]] || error "ERROR: config-dir not properly used"

beforeEach

# test configure w/ custom config-dir AND custom configuration
mkdir ./temp-config-dir
"$DOPPLER_BINARY" configure set config 123 --config-dir=./temp-config-dir --configuration ./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --plain)"
[[ "$config" == "123" ]] || error "ERROR: configuration not properly used when specified with config-dir"

beforeEach

# test unset
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
"$DOPPLER_BINARY" configure unset config --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":""}' ]] || error "ERROR: unexpected config contents after 'unset'"

beforeEach

# test unset wrong scope
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
"$DOPPLER_BINARY" configure unset config --configuration=./temp-config --scope=/otherscope --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: unexpected config contents after 'unset' differnet scope"

beforeEach

# test set overwriting existing value
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
"$DOPPLER_BINARY" configure set config 456 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"456"}' ]] || error "ERROR: unexpected config contents after double 'set'"

beforeEach

# test set with equals sign (e.g. 'set foo=bar')
"$DOPPLER_BINARY" configure set config=123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: unexpected config contents after 'set' w/ equals sign"

beforeEach

# test set using stdin
echo 123 | "$DOPPLER_BINARY" configure set config --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: unexpected config contents after 'set' w/ stdin"

beforeEach

#
# CLI v3 compatability tests (DPLR-435)
#

# test setting config and reading enclave.config
"$DOPPLER_BINARY" configure set config 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get enclave.config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: unexpected config contents after double 'set'"

beforeEach

# test setting enclave.config and reading config
"$DOPPLER_BINARY" configure set enclave.config 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get config --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.config":"123"}' ]] || error "ERROR: unexpected config contents after double 'set'"

beforeEach

# test setting project and reading enclave.project
"$DOPPLER_BINARY" configure set project 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get enclave.project --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.project":"123"}' ]] || error "ERROR: unexpected config contents after double 'set'"

beforeEach

# test setting enclave.project and reading project
"$DOPPLER_BINARY" configure set enclave.project 123 --configuration=./temp-config --scope=/ --silent
config="$("$DOPPLER_BINARY" configure get project --configuration=./temp-config --scope=/ --json)"
[[ "$config" == '{"enclave.project":"123"}' ]] || error "ERROR: unexpected config contents after double 'set'"

afterAll
