# Doppler CLI

## Warning: This tool is current pre-release. For the current stable version, please use the [node-cli](https://github.com/DopplerHQ/node-cli).

The Doppler CLI is the official tool for interacting with your Enclave secrets and configuration.

**You can:**

- Manage your secrets, projects, and environments
- View activity and audit logs
- Execute applications with your secrets injected into the environment

## Install

The Doppler CLI is available in several popular package managers. It's also available as a standalone binary on the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page.

### macOS

Using brew is recommended:

```sh
$ brew install dopplerhq/cli/doppler
$ doppler --version
```

To update:
```sh
$ brew upgrade doppler
```

Alternatively, you can install the doppler `pkg` file from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Note that this installation method does not support easy updates. To update, you'll need to install the new `pkg` file.

### Windows

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

#### Debian/Ubuntu (apt)

```sh
# add Bintray's GPG key
$ sudo apt-key adv --keyserver pool.sks-keyservers.net --recv-keys 379CE192D401AB61

# add Doppler's apt repo
$ sudo echo "deb https://dl.bintray.com/dopplerhq/doppler-deb stable main" > /etc/apt/sources.list.d/dopplerhq-doppler-deb.list

# fetch latest packages
$ sudo apt-get update

# install doppler cli
$ sudo apt-get install doppler

# execute the cli
$ doppler --version
```

To update:

```sh
$ sudo apt-get update && sudo apt-get upgrade doppler
```

#### RedHat/CentOS (yum)

```sh
# add Doppler's yum repo
$ sudo wget https://bintray.com/dopplerhq/doppler-rpm/rpm -O /etc/yum.repos.d/bintray-dopplerhq-doppler-rpm.repo

# fetch and update latest packages
$ sudo yum update

# install doppler cli
$ sudo yum install doppler

# execute the cli
$ doppler --version
```

To update:

```sh
$ sudo yum update doppler
```

### Docker

Docker containers are currently built using two base images: `alpine` and `node:lts-alpine`.

Example:

```sh
$ docker run --rm -it dopplerhq/cli --version
v1.0.0
$ docker run --rm -it dopplerhq/cli:node --version
v1.0.0
```

Here's an example Dockerfile showing how you can build on top of Doppler's base images:

```dockerfile
FROM dopplerhq/cli:node

# doppler args are passed at runtime
ENV DOPPLER_API_KEY="" DOPPLER_PROJECT="" DOPPLER_CONFIG=""

COPY . .

ENTRYPOINT doppler run --key="$DOPPLER_API_KEY" --project="$DOPPLER_PROJECT" --config="$DOPPLER_CONFIG" -- node index.js
```

### Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb` and `.rpm` packages. If a binary doesn't exist for the OS/architecture you use, please open a GitHub Issue.

## Usage

Once installed, you can access the Doppler CLI with the `doppler` command.

```sh
$ doppler configure set key=$YOUR_API_KEY  # set local credentials
$ doppler setup                            # select your project and config
$ doppler configure --all                  # (optional) view local configuration
```

The first command will save your api key to the local configuration file, and it will be scoped to the current directory. You can modify this scope by specifying the `--scope` flag. See `doppler help configure set` for more info, or run `doppler configure --all` to view your current configuration.

For a list of all commands:

```sh
$ doppler help
```

## Development

### Build

```sh
$ make build
$ ./doppler --version
```

### Test

Build for all release targets:

```
$ make test-release
```

### Release

To release a new version, run:

```
$ make release V=vX.Y.Z
```

This command will push local changes to Origin, create a new tag, and push the tag to Origin. It will then build and release the doppler binaries.

Note: The release will automatically fail if the tag and HEAD have diverged:

`   ⨯ release failed after 0.13s error=git tag v0.0.2 was not made against commit c9c6950d18790c17db11fedae331a226f8f12c6b`

### Help

**Issue**: `gpg: signing failed: Inappropriate ioctl for device`

- **Fix**: `export GPG_TTY=$(tty)`

**Issue**: After releasing, your personal account is logged out of the docker daemon

- **Fix**: Log in again with this registry manually specified: `docker login https://docker.io`

- **Why**: The release script explicitly scopes the `dopplerbot` docker login to `https://index.docker.io/v1/`. By explicitly scoping your personal login, you ensure these two logins don't conflict (and thus your personal login doesn't get removed on script cleanup). If not specified, `docker` treats these two registries as aliases.


#### Generate a GPG key

Store the keys and passphrase in your enclave config

```
$ gpg --full-generate-key
$ gpg --list-secret-keys  # copy the key's 40-character ID
$ gpg --armor --export-secret-key KEY_ID
$ gpg --armor --export KEY_ID
$ gpg --keyserver pgp.mit.edu --send-key LAST_8_DIGITS_OF_KEY_ID
```
