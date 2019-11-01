# go-cli-test


## Build

Build the app with `go build`. Run `build.sh` instead if you want to build cross-compiled binaries.

## Run

#### Local development

`./doppler-run -environment=$(doppler config:get environment) -pipeline=$(doppler config:get pipeline) -key=$(doppler config:get key)  -- YOUR COMMAND HERE`

#### Docker (alpine):

`docker run --rm -it -v "$(pwd)/bin":/mnt/bin -e key=$(doppler config:get key) -e pipeline=$(doppler config:get pipeline) -e environment=$(doppler config:get environment) alpine:latest`


## Release

```
git status # confirm clean workspace and no unpushed changes
git tag vX.Y.Z -a -m "Release notes"
git push --tags
GITHUB_TOKEN=$(doppler secrets get GITHUB_TOKEN --plain goreleaser release --rm-dist
```

Note: The release will automatically fail if the tag and HEAD have diverged
`   ⨯ release failed after 0.13s error=git tag v0.0.2 was not made against commit c9c6950d18790c17db11fedae331a226f8f12c6b`
