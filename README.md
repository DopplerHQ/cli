# Doppler CLI

## Warning: This tool is current pre-release. For the current stable version, please use the [node-cli](https://github.com/DopplerHQ/node-cli).

The Doppler CLI is the official tool for interacting with your Enclave secrets and configuration.

**You can:**

- Manage your secrets, projects, and environments
- View activity and audit logs
- Execute applications with your secrets injected into the environment

## Install

The Doppler CLI is available in several popular package managers. It's also [available](https://github.com/DopplerHQ/cli/releases/latest) as a standalone binary.

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

Alternatively, you can download the doppler `pkg` file from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. This will install the doppler binary in `/usr/local/bin`. Note that this installation method does not support seamless updates. To update, you'll need to download and run the new `pkg` file.

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

#### Debian/Ubuntu (apt)

```sh
# add Bintray's GPG key
$ sudo apt-key adv --keyserver pool.sks-keyservers.net --recv-keys 379CE192D401AB61

# add Doppler's apt repo
$ sudo echo "deb https://dl.bintray.com/dopplerhq/doppler-deb stable main" > /etc/apt/sources.list.d/dopplerhq-doppler.list

# fetch and install latest doppler cli
$ sudo apt-get update && sudo apt-get install doppler

# (optional) print cli version
$ doppler --version
```

To update:

```sh
$ sudo apt-get update && sudo apt-get upgrade doppler
```

#### RedHat/CentOS (yum)

```sh
# add Doppler's yum repo
$ sudo wget https://bintray.com/dopplerhq/doppler-rpm/rpm -O /etc/yum.repos.d/bintray-dopplerhq-doppler.repo

# update packages and install latest doppler cli
$ sudo yum update && sudo yum install doppler

# (optional) print cli version
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
ENV DOPPLER_TOKEN="" DOPPLER_PROJECT="" DOPPLER_CONFIG=""

COPY . .

ENTRYPOINT doppler run --token="$DOPPLER_TOKEN" --project="$DOPPLER_PROJECT" --config="$DOPPLER_CONFIG" -- node index.js
```

### Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb` and `.rpm` packages. If a binary doesn't exist for the OS/architecture you use, please open a GitHub Issue.

## Usage

Once installed, setup should only take a minute. You'll authorize the CLI to access your Doppler worplace, and then select your project and config.

```sh
$ doppler login                     # generate auth credentials
$ doppler setup                     # select your project and config
# optional
$ doppler configure --all           # view local configuration
```

By default, `doppler login` and `doppler setup` will scope your configuration to the current directory. You can modify the scope by specifying the `--scope` flag. Run `doppler help` for more information.

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
