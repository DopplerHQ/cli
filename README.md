# Doppler CLI

The Doppler CLI is the official tool for interacting with your Enclave secrets and configuration.

**You can:**

- Manage your secrets, projects, and environments
- View activity and audit logs
- Execute applications with your secrets injected into the environment

## Install

The Doppler CLI is available in several popular package managers. It can also be installed via [shell script](https://github.com/DopplerHQ/cli/blob/master/INSTALL.md#linuxmacosbsd-shell-script), and downloaded as a [standalone binary](https://github.com/DopplerHQ/cli/releases/latest).

For more info, including instructions on verifying binary signatures, see the [Install](INSTALL.md) page.

### macOS

Using [brew](https://brew.sh/) is recommended:

```sh
$ brew install dopplerhq/cli/doppler
$ doppler --version
```

To update:
```sh
$ brew upgrade doppler
```

For installation without brew, see the [Install](INSTALL.md) page.

### Windows

Using [scoop](https://scoop.sh/) is recommended:

```sh
$ scoop bucket add doppler https://github.com/DopplerHQ/scoop-doppler.git
$ scoop install doppler
$ doppler --version
```

To update:

```sh
$ scoop update doppler
```

### Linux

See [Install](INSTALL.md) page for instructions.

### Docker

We currently build these Docker images:
- `dopplerhq/cli` based on `alpine`
- `dopplerhq/cli:node` based on `node:lts-alpine`
- `dopplerhq/cli:python` based on `python:3-alpine`
- `dopplerhq/cli:ruby` based on `ruby:2-alpine`

For more info, see the [Install](INSTALL.md) page.

## Usage

Once installed, setup should only take a minute. You'll authorize the CLI to access your Doppler workplace, and then select your project and config.

```sh
$ doppler login                     # generate auth credentials
$ doppler enclave setup             # select your project and config
# optional
$ doppler configure --all           # view local configuration
```

By default, `doppler login` scopes the generated token globally (`--scope=*`). This means that the token will be accessible to your projects in any local directory. To limit the scope of the token, specify the `scope` flag during login: `doppler login --scope=.`.

Enclave setup (i.e. `doppler enclave setup`) scopes the enclave project and config to the current directory (`--scope=.`). You can also modify this scope with the `scope` flag. Run `doppler help` for more information.

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

Store the keys and passphrase in your enclave config

```
$ gpg --full-generate-key
$ gpg --list-secret-keys  # copy the key's 40-character ID
$ gpg --armor --export-secret-key KEY_ID
$ gpg --armor --export KEY_ID
$ gpg --keyserver keyserver.ubuntu.com --send-key LAST_8_DIGITS_OF_KEY_ID
```
