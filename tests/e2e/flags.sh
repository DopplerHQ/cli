#!/bin/bash

set -euo pipefail

TEST_NAME="flags"
TEST_CONFIG_DIR="./temp-config-dir"

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
  rm -rf $TEST_CONFIG_DIR
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

flags=('analytics' 'env-warning' 'update-check')

beforeAll

beforeEach

# verify defaults
for flag in "${flags[@]}"; do
  [[ "$("$DOPPLER_BINARY" configure flags get "$flag" --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect default for $flag"
done

beforeEach

# verify set/get
for flag in "${flags[@]}"; do
  "$DOPPLER_BINARY" configure flags disable "$flag" --config-dir=$TEST_CONFIG_DIR >/dev/null 2>/dev/null
  [[ "$("$DOPPLER_BINARY" configure flags get "$flag" --plain --config-dir=$TEST_CONFIG_DIR)" == 'false' ]] || error "ERROR: incorrect value for $flag after disabling"
  "$DOPPLER_BINARY" configure flags enable "$flag" --config-dir=$TEST_CONFIG_DIR >/dev/null 2>/dev/null
  [[ "$("$DOPPLER_BINARY" configure flags get "$flag" --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect value for $flag after enabling"
done

# beforeEach

# verify reset
for flag in "${flags[@]}"; do
  "$DOPPLER_BINARY" configure flags disable "$flag" --config-dir=$TEST_CONFIG_DIR >/dev/null 2>/dev/null
  "$DOPPLER_BINARY" configure flags reset -y "$flag" --config-dir=$TEST_CONFIG_DIR >/dev/null 2>/dev/null
  [[ "$("$DOPPLER_BINARY" configure flags get "$flag" --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect value for $flag after reset"
done

beforeEach

# verify interoperability between 'flags' command and legacy 'analytics' command
[[ "$("$DOPPLER_BINARY" configure flags get analytics --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect initial value for analytics"
"$DOPPLER_BINARY" analytics disable --config-dir=$TEST_CONFIG_DIR >/dev/null 2>&1
[[ "$("$DOPPLER_BINARY" configure flags get analytics --plain --config-dir=$TEST_CONFIG_DIR)" == 'false' ]] || error "ERROR: incorrect disabled value for analytics"
"$DOPPLER_BINARY" analytics enable --config-dir=$TEST_CONFIG_DIR >/dev/null 2>&1
[[ "$("$DOPPLER_BINARY" configure flags get analytics --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect enabled value for analytics"

beforeEach

# parse legacy analytics field from config file
mkdir ./temp-config-dir
cat << EOF > ./temp-config-dir/.doppler.yaml
analytics:
    disable: true
EOF

[[ "$("$DOPPLER_BINARY" configure flags get analytics --plain --config-dir=$TEST_CONFIG_DIR)" == 'false' ]] || error "ERROR: incorrect value read when parsing legacy analytics field in config file"

cat << EOF > ./temp-config-dir/.doppler.yaml
analytics:
    disable: false
EOF

[[ "$("$DOPPLER_BINARY" configure flags get analytics --plain --config-dir=$TEST_CONFIG_DIR)" == 'true' ]] || error "ERROR: incorrect value read when parsing legacy analytics field in config file"

afterAll
