## Development

To build the Doppler CLI for development, you'll need Golang installed.

### Build

```sh
$ make build
$ ./doppler --version
```

### Test

Testing building for all release targets:

```
$ make test-release
```

### Release

To release a new version, run this command with `$NEW_VERSION` set to one of `major`, `minor`, `patch`, or `vX.Y.Z` (where `vX.Y.Z` is a valid semantic version).

```
$ make release V=$NEW_VERSION
```

This command will push local changes to origin, create a new tag, and push the tag to origin. It will then build and release the doppler binaries.

Note: The release will automatically fail if the tag and HEAD have diverged:

`   ⨯ release failed after 0.13s error=git tag v0.0.2 was not made against commit c9c6950d18790c17db11fedae331a226f8f12c6b`

Note: In the goreleaser output, it will state that artifact signing is disabled. This is due to the custom args we pass goreleaser (so that we can specify our GPG key). You can verify that signing works by the presence of a `checksums.txt.sig` file.

### Help

**Issue**: `gpg: signing failed: Inappropriate ioctl for device`

- **Fix**: `export GPG_TTY=$(tty)`

**Issue**: After releasing, your personal account is logged out of the docker daemon

- **Fix**: Log in again with this registry manually specified: `docker login https://docker.io`

- **Why**: The release script explicitly scopes the `dopplerbot` docker login to `https://index.docker.io/v1/`. By explicitly scoping your personal login, you ensure these two logins do not conflict (and thus your personal login does not get removed on script cleanup). If not specified, `docker` treats these two registries as aliases.


#### Generate a GPG key

Store the keys and passphrase in your Doppler config

```
$ gpg --full-generate-key
$ gpg --list-secret-keys  # copy the key's 40-character ID
$ gpg --armor --export-secret-key KEY_ID
$ gpg --armor --export KEY_ID
$ gpg --keyserver keyserver.ubuntu.com --send-key LAST_8_DIGITS_OF_KEY_ID
```
