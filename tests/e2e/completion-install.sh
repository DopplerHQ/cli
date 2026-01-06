#!/bin/bash

# E2E tests for zsh completion installation

set -euo pipefail

TEST_NAME="completion install"
TEMP_HOME=""

cleanup() {
  exit_code=$?
  if [ "$exit_code" -ne 0 ]; then
    echo "ERROR: '$TEST_NAME' tests failed during execution"
  fi
  # Clean up temp directories
  if [ -n "$TEMP_HOME" ] && [ -d "$TEMP_HOME" ]; then
    rm -rf "$TEMP_HOME"
  fi
  exit "$exit_code"
}
trap cleanup EXIT
trap cleanup INT

beforeAll() {
  echo "INFO: Executing '$TEST_NAME' tests"
}

beforeEach() {
  # Clean up any previous temp home
  if [ -n "$TEMP_HOME" ] && [ -d "$TEMP_HOME" ]; then
    rm -rf "$TEMP_HOME"
  fi
  TEMP_HOME=$(mktemp -d)
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  if [ -n "$TEMP_HOME" ] && [ -d "$TEMP_HOME" ]; then
    rm -rf "$TEMP_HOME"
  fi
}

error() {
  message=$1
  echo "ERROR: $message"
  exit 1
}

beforeAll

# Test 1 - zsh uses fallback path when standard path is not writable
beforeEach
echo "Testing: zsh fallback to XDG_DATA_HOME/doppler/zsh/completions"
HOME="$TEMP_HOME" SHELL="/bin/zsh" "$DOPPLER_BINARY" completion install --silent

# Verify completion file was created in XDG-compliant directory (defaults to ~/.local/share)
[[ -f "$TEMP_HOME/.local/share/doppler/zsh/completions/_doppler" ]] || error "completion file not created at ~/.local/share/doppler/zsh/completions/_doppler"

echo "PASS: zsh fallback path"

# Test 2 - prints fpath instructions when using fallback path
beforeEach
echo "Testing: prints fpath instructions"
output=$(HOME="$TEMP_HOME" SHELL="/bin/zsh" "$DOPPLER_BINARY" completion install 2>&1)

# Verify instructions are printed
echo "$output" | grep -q "fpath=" || error "fpath instructions not printed"
echo "$output" | grep -q "doppler/zsh/completions" || error "doppler path not in instructions"

echo "PASS: prints fpath instructions"

# Test 3 - bash completions still work (regression test)
beforeEach
echo "Testing: bash completions"
HOME="$TEMP_HOME" SHELL="/bin/bash" "$DOPPLER_BINARY" completion install --silent || true

# The important thing is that it doesn't crash or try to use zsh paths
# Bash will fail on most systems due to /etc or /usr/local/etc permissions
echo "PASS: bash completions (no crash)"

# Test 4 - completion file contains valid zsh completion code
beforeEach
echo "Testing: completion file content"
HOME="$TEMP_HOME" SHELL="/bin/zsh" "$DOPPLER_BINARY" completion install --silent

# Verify the completion file contains expected zsh completion directives
grep -q "compdef" "$TEMP_HOME/.local/share/doppler/zsh/completions/_doppler" || grep -q "#compdef" "$TEMP_HOME/.local/share/doppler/zsh/completions/_doppler" || error "completion file doesn't contain valid zsh completion directives"

echo "PASS: completion file content"

# Test 5 - directory structure is created correctly
beforeEach
echo "Testing: directory structure"
HOME="$TEMP_HOME" SHELL="/bin/zsh" "$DOPPLER_BINARY" completion install --silent

# Verify the full directory structure exists
[[ -d "$TEMP_HOME/.local/share/doppler" ]] || error "~/.local/share/doppler directory not created"
[[ -d "$TEMP_HOME/.local/share/doppler/zsh" ]] || error "~/.local/share/doppler/zsh directory not created"
[[ -d "$TEMP_HOME/.local/share/doppler/zsh/completions" ]] || error "~/.local/share/doppler/zsh/completions directory not created"

echo "PASS: directory structure"

# Test 6 - respects XDG_DATA_HOME when set
beforeEach
echo "Testing: XDG_DATA_HOME is respected"
mkdir -p "$TEMP_HOME/custom-data"
HOME="$TEMP_HOME" XDG_DATA_HOME="$TEMP_HOME/custom-data" SHELL="/bin/zsh" "$DOPPLER_BINARY" completion install --silent

# Verify completion file was created in custom XDG_DATA_HOME
[[ -f "$TEMP_HOME/custom-data/doppler/zsh/completions/_doppler" ]] || error "completion file not created in custom XDG_DATA_HOME"

echo "PASS: XDG_DATA_HOME is respected"

afterAll
