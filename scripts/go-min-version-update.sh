#!/bin/bash

# From https://go.dev/doc/devel/release:
#
# > Each major Go release is supported until there are two newer major releases.
# > For example, Go 1.5 was supported until the Go 1.7 release, and Go 1.6 was
# > supported until the Go 1.8 release. We fix critical problems, including critical
# > security problems, in supported releases as needed by issuing minor revisions
# > (for example, Go 1.6.1, Go 1.6.2, and so on).

SALUS_CONFIG_PATH="${GITHUB_WORKSPACE:-..}/salus-config.yaml"

# Get the latest stable release of the previous release from current (i.e., if
# the most recent Go release is 1.26.0, fetch the most recent 1.25.x version).
new_target_min_version=$(curl -s https://go.dev/dl/?mode=json | jq -r '.[-1] | .version | gsub("go";"")')

# Extract the minimum Go version from the Salus config to compare against. This
# is in YAML format, so we're just brute forcing with grep and awk.
current_target_min_version=$(grep -A2 "GoVersionScanner:" "${SALUS_CONFIG_PATH}" | grep min_version | awk '{print $2}' | tr -d "['\"]")

# Compare both versions. We don't really care which is "newer" than the other
# because we trust what's coming from Go's API, so if they don't match we just
# update to what we got from Go's API using sed. Note that `-i ''` is used to
# work properly on MacOS.
if [ "$new_target_min_version" != "$current_target_min_version" ]; then
  sed -i '' "/min_version/s/$current_target_min_version/$new_target_min_version/" "${SALUS_CONFIG_PATH}"
fi
