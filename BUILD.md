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

To release a new version, run the [release](https://github.com/DopplerHQ/cli/actions/workflows/release.yaml) GitHub Action manually and specify whether you want to bump the major, minor, or patch version.
Note that this will require approval from a CLI admin before the action will be allowed to run.

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
