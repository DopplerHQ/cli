#!/bin/bash

echo "$GOOGLE_CREDS" > "$GOOGLE_APPLICATION_CREDENTIALS"
goreleaser release --rm-dist
rm "$GOOGLE_APPLICATION_CREDENTIALS"
