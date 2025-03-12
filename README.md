# Doppler CLI

The Doppler CLI is the official tool for interacting with your Doppler secrets and configuration.

**You can:**

- Manage your secrets, projects, and environments
- Execute applications with your secrets injected into the environment
- View activity and audit logs

## Install

The Doppler CLI is available in several popular package managers. It can also be installed via [shell script](https://github.com/DopplerHQ/cli/blob/master/INSTALL.md#shell-script), [GitHub Action](https://github.com/DopplerHQ/cli-action), or downloaded as a [standalone binary](https://github.com/DopplerHQ/cli/releases/latest).

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

For installation without brew, see the [Install](INSTALL.md#macos) page.

### Windows

Using winget is recommended:

```sh
$ winget install doppler
$ doppler --version
```

To update:

```sh
$ winget upgrade doppler
```

For additional options, see the [Install](INSTALL.md#windows) page.

### Shell script

This option is recommend for CI jobs and other environments that won't make use of package manager updates.

```sh
$ (curl -Ls --tlsv1.2 --proto "=https" --retry 3 https://cli.doppler.com/install.sh || wget -t 3 -qO- https://cli.doppler.com/install.sh) | sh
```

For more info, see the [Install](INSTALL.md#shell-script) page.

### Linux

See [Install](INSTALL.md#linux) page for instructions.

### Docker

See [Install](INSTALL.md#docker) page for instructions.

### GitHub Action

You can install the latest version of the CLI via GitHub Action. See the cli-action [repo](https://github.com/DopplerHQ/cli-action) for more info.

## Usage

Setup should only take a minute (literally). You'll authorize the CLI to access your Doppler workplace, and then select your project and config.

```sh
$ doppler login                     # generate auth credentials
$ doppler setup                     # select your project and config
# optional
$ doppler configure --all           # view local configuration
```

By default, `doppler login` scopes the auth token to the root directory (`--scope=/`). This means that the token will be accessible to projects using the Doppler CLI in any subdirectory. To limit this, specify the `scope` flag during login: `doppler login --scope=./` or `doppler login --scope ~/projects/backend`.

Setup (i.e. `doppler setup`) scopes the selected project and config to the current directory (`--scope=./`). You can also modify this scope with the `scope` flag. Run `doppler help` for more information.

## Go Version

This project defines its Go version in a number of places. If updating the Go version, search for `GO_VERSION_DEF` and ensure that all locations are updated.
