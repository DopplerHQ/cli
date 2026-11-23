#!/bin/bash

set -euo pipefail

TEST_NAME="run-mount"

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
  rm -f ./secrets.json ./secrets.env
}

afterAll() {
  echo "INFO: Completed '$TEST_NAME' tests"
  "$DOPPLER_BINARY" run clean --max-age=0s --silent
  rm -f ./secrets.json ./secrets.env
}

beforeAll

beforeEach

# verify content of mounted secrets file
EXPECTED_SECRETS='{"DOPPLER_CONFIG":"e2e","DOPPLER_ENVIRONMENT":"e2e","DOPPLER_PROJECT":"cli","FOO":"bar","HOME":"123","TEST":"abc"}'
actual="$("$DOPPLER_BINARY" run --mount secrets.json --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [ "$actual" != "$EXPECTED_SECRETS" ]; then
 echo "ERROR: mounted secrets file has invalid contents"
 exit 1
fi

beforeEach

# verify secrets aren't injected into environment
# this will succeed
"$DOPPLER_BINARY" run --command "printenv DOPPLER_ENVIRONMENT" && \
  # this should fail
  "$DOPPLER_BINARY" run --mount secrets.json --command "printenv DOPPLER_ENVIRONMENT" && \
  (echo "ERROR: secrets injected into environment despite using mounted secrets file" && exit 1)

beforeEach

# verify specified mount path is used and converted to absolute path
expected="$(pwd)/secrets.json"
actual="$("$DOPPLER_BINARY" run --mount secrets.json --command "echo \$DOPPLER_CLI_SECRETS_PATH")"
if [ "$actual" != "$expected" ]; then
 echo "ERROR: secrets are not mounted to specified path"
 exit 1
fi

beforeEach

# verify format is auto detected
EXPECTED_SECRETS='DOPPLER_CONFIG="e2e"\nDOPPLER_ENVIRONMENT="e2e"\nDOPPLER_PROJECT="cli"\nFOO="bar"\nHOME="123"\nTEST="abc"'
actual="$("$DOPPLER_BINARY" run --mount secrets.env --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted secrets file with auto-detected env format has invalid contents"
 exit 1
fi

beforeEach

# verify fallback file can be used
EXPECTED_SECRETS='DOPPLER_CONFIG="e2e"\nDOPPLER_ENVIRONMENT="e2e"\nDOPPLER_PROJECT="cli"\nFOO="bar"\nHOME="123"\nTEST="abc"'
"$DOPPLER_BINARY" run --mount secrets.env --fallback ./fallback.txt --command "cat \$DOPPLER_CLI_SECRETS_PATH" > /dev/null
actual="$("$DOPPLER_BINARY" run --mount secrets.env --fallback ./fallback.txt --fallback-only --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
rm -f ./fallback.txt
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted secrets file with env format has invalid contents"
 exit 1
fi

beforeEach

# verify attempting to mount with mismatched format without --fallback-only gracefully handles mismatch and fetches correct format
EXPECTED_SECRETS='{"DOPPLER_CONFIG":"e2e","DOPPLER_ENVIRONMENT":"e2e","DOPPLER_PROJECT":"cli","FOO":"bar","HOME":"123","TEST":"abc"}'
"$DOPPLER_BINARY" run --mount secrets.env --format env --fallback ./fallback.txt --command "cat \$DOPPLER_CLI_SECRETS_PATH" > /dev/null
actual="$("$DOPPLER_BINARY" run --mount secrets.json --format json --fallback ./fallback.txt --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
rm -f ./fallback.txt
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted secrets file with json format has invalid contents"
 exit 1
fi

beforeEach

# verify --only-secrets works with a mount format that isn't json (the DOPPLER_ secrets are always included)
EXPECTED_SECRETS='DOPPLER_CONFIG="e2e"\nDOPPLER_ENVIRONMENT="e2e"\nDOPPLER_PROJECT="cli"\nFOO="bar"'
actual="$("$DOPPLER_BINARY" run --mount secrets.env --only-secrets FOO --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted env secrets file with --only-secrets has invalid contents"
 exit 1
fi

beforeEach

# verify specified format is used (--mount-format, deprecated)
EXPECTED_SECRETS='{"DOPPLER_CONFIG":"e2e","DOPPLER_ENVIRONMENT":"e2e","DOPPLER_PROJECT":"cli","FOO":"bar","HOME":"123","TEST":"abc"}'
actual="$("$DOPPLER_BINARY" run --mount secrets.env --mount-format json --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted secrets file with json format has invalid contents"
 exit 1
fi

beforeEach

# verify specified format is used (--format, new flag)
EXPECTED_SECRETS='{"DOPPLER_CONFIG":"e2e","DOPPLER_ENVIRONMENT":"e2e","DOPPLER_PROJECT":"cli","FOO":"bar","HOME":"123","TEST":"abc"}'
actual="$("$DOPPLER_BINARY" run --mount secrets.env --format json --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted secrets file with --format json has invalid contents"
 exit 1
fi

beforeEach

# verify --preserve-env with a list merges only the listed env vars into a mounted json file
EXPECTED_SECRETS='{"DOPPLER_CONFIG":"e2e","DOPPLER_ENVIRONMENT":"e2e","DOPPLER_PROJECT":"cli","FOO":"from-env","HOME":"123","LISTED":"from-env","TEST":"abc"}'
actual="$(FOO=from-env LISTED=from-env UNLISTED=from-env "$DOPPLER_BINARY" run --mount secrets.json --preserve-env=FOO,LISTED --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$EXPECTED_SECRETS" ]]; then
 echo "ERROR: mounted secrets file with --preserve-env list has invalid contents"
 exit 1
fi

beforeEach

# verify --preserve-env=true merges the whole env into a mounted json file, as it does for env injection
actual="$(FOO=from-env UNLISTED=from-env "$DOPPLER_BINARY" run --mount secrets.json --preserve-env --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
for expected in '"FOO":"from-env"' '"UNLISTED":"from-env"' '"TEST":"abc"' "\"HOME\":\"$HOME\""; do
  if [[ "$actual" != *"$expected"* ]]; then
   echo "ERROR: mounted secrets file with --preserve-env=true is missing $expected"
   exit 1
  fi
done

beforeEach

# verify --preserve-env overrides matching secrets in a mounted template
actual="$(FOO=from-env "$DOPPLER_BINARY" run --mount secrets.json --mount-template /dev/stdin --preserve-env=FOO --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.FOO}}')"
if [[ "$actual" != "from-env" ]]; then
 echo "ERROR: mounted secrets template with --preserve-env has invalid contents"
 exit 1
fi

beforeEach

# verify specified name transformer is used
EXPECTED_SECRETS='{"TF_VAR_doppler_config":"e2e","TF_VAR_doppler_environment":"e2e","TF_VAR_doppler_project":"cli","TF_VAR_foo":"bar","TF_VAR_home":"123","TF_VAR_test":"abc"}'
actual="$("$DOPPLER_BINARY" run --mount secrets.json --name-transformer tf-var --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$EXPECTED_SECRETS" ]]; then
 echo "ERROR: mounted secrets file with name transformer has invalid contents"
 exit 1
fi

beforeEach

# verify template is used
EXPECTED_SECRETS='e2e'
actual="$("$DOPPLER_BINARY" run --mount secrets.json --mount-template /dev/stdin --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.DOPPLER_CONFIG}}')"
if [[ "$actual" != "$EXPECTED_SECRETS" ]]; then
 echo "ERROR: mounted secrets file with template has invalid contents"
 exit 1
fi

beforeEach

# verify --mount-template can be used with --mount-format=template (deprecated flag)
EXPECTED_SECRETS='e2e'
actual="$("$DOPPLER_BINARY" run --mount secrets.json --mount-template /dev/stdin --mount-format template --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.DOPPLER_CONFIG}}')"
if [[ "$actual" != "$EXPECTED_SECRETS" ]]; then
 echo "ERROR: mounted secrets file with template has invalid contents"
 exit 1
fi

beforeEach

# verify --mount-template can be used with --format=template (new flag)
EXPECTED_SECRETS='e2e'
actual="$("$DOPPLER_BINARY" run --mount secrets.json --mount-template /dev/stdin --format template --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.DOPPLER_CONFIG}}')"
if [[ "$actual" != "$EXPECTED_SECRETS" ]]; then
 echo "ERROR: mounted secrets file with --format template has invalid contents"
 exit 1
fi

beforeEach

# verify --mount-template cannot be used with --format=json
"$DOPPLER_BINARY" run --mount secrets.json --mount-template /dev/stdin --format json --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.DOPPLER_CONFIG}}' && \
  (echo "ERROR: mounted secrets with template was successful with invalid --format" && exit 1)

beforeEach

# verify --mount-template cannot be used without --mount
"$DOPPLER_BINARY" run --mount-template /dev/stdin --command "cat \$DOPPLER_CLI_SECRETS_PATH" <<<'{{.DOPPLER_CONFIG}}' && \
  (echo "ERROR: mounted secrets with template was successful without --mount" && exit 1)

beforeEach

# verify existing env value is ignored when --preserve-env is specified with a format that doesn't support it
EXPECTED_SECRETS='DOPPLER_CONFIG="e2e"\nDOPPLER_ENVIRONMENT="e2e"\nDOPPLER_PROJECT="cli"\nFOO="bar"\nHOME="123"\nTEST="abc"'
actual="$(DOPPLER_CONFIG="test" "$DOPPLER_BINARY" run --preserve-env --config e2e --mount secrets.env --command "cat \$DOPPLER_CLI_SECRETS_PATH")"
if [[ "$actual" != "$(echo -e "$EXPECTED_SECRETS")" ]]; then
 echo "ERROR: mounted env secrets file with --preserve-env has invalid contents"
 exit 1
fi

afterAll
