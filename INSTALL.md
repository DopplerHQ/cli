# Install

The Doppler CLI is available in several popular package managers. It's also [available](https://github.com/DopplerHQ/cli/releases/latest) as a standalone binary.

## macOS

Using [brew](https://brew.sh/) is recommended:

```sh
$ brew install dopplerhq/cli/doppler
$ doppler --version
```

To update:
```sh
$ brew upgrade doppler
```

Alternatively, you can install the CLI via [shell script](#linuxmacosbsd-shell-script), or via the doppler `.pkg` file on the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. These methods will install the doppler binary directly to `/usr/local/bin` and do not support seamless updates. To update, you'll need to re-run the installation.

## Windows

### Winget

Using winget is recommended:

```sh
$ winget install doppler
$ doppler --version
```

To update:

```sh
$ winget upgrade doppler
```

### Scoop

Using [scoop](https://scoop.sh/) is supported:

```sh
$ scoop bucket add doppler https://github.com/DopplerHQ/scoop-doppler.git
$ scoop install doppler
$ doppler --version
```

To update:

```sh
$ scoop update doppler
```

### Git Bash

Using [Git Bash](https://git-scm.com/download/win) is also supported:

```sh
$ mkdir -p $HOME/bin
$ curl -Ls --tlsv1.2 --proto "=https" --retry 3 https://cli.doppler.com/install.sh | sh -s -- --install-path $HOME/bin
$ doppler --version
```

## Linux

### Alpine (apk)

```sh
# add Doppler's RSA key
$ wget -q -t3 'https://packages.doppler.com/public/cli/rsa.8004D9FF50437357.key' -O /etc/apk/keys/cli@doppler-8004D9FF50437357.rsa.pub

# add Doppler's apk repo
$ echo 'https://packages.doppler.com/public/cli/alpine/any-version/main' | tee -a /etc/apk/repositories

# fetch and install latest doppler cli
$ apk add doppler

# (optional) print cli version
$ doppler --version
```

To update:

```sh
$ apk upgrade doppler
```

### Debian/Ubuntu (apt)

```sh
# install pre-reqs
$ apt-get update && apt-get install -y apt-transport-https ca-certificates curl gnupg sudo

# add Doppler's GPG key
$ curl -sLf --retry 3 --tlsv1.2 --proto "=https" 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key' | gpg --dearmor | sudo tee /etc/apt/keyrings/doppler.gpg >/dev/null

# add Doppler's apt repo
$ echo "deb [signed-by=/etc/apt/keyrings/doppler.gpg] https://packages.doppler.com/public/cli/deb/debian any-version main" | sudo tee /etc/apt/sources.list.d/doppler-cli.list

# fetch and install latest doppler cli
$ sudo apt-get update && sudo apt-get install doppler

# (optional) print cli version
$ doppler --version
```

To update:

```sh
$ sudo apt-get update && sudo apt-get upgrade doppler
```

### RedHat/CentOS (yum)

```sh
# add Doppler's GPG key
$ sudo rpm --import 'https://packages.doppler.com/public/cli/gpg.DE2A7741A397C129.key'

# add Doppler's yum repo
$ curl -sLf --retry 3 --tlsv1.2 --proto "=https" 'https://packages.doppler.com/public/cli/config.rpm.txt' | sudo tee /etc/yum.repos.d/doppler-cli.repo

# update packages and install latest doppler cli
$ sudo yum update && sudo yum install doppler

# (optional) print cli version
$ doppler --version
```

To update:

```sh
$ sudo yum update doppler
```

## Shell script

You can bypass package managers and quickly install the latest version of the CLI via shell script. The script automatically downloads and installs the CLI binary most appropriate for your system's architecture. It is also fully POSIX compliant to support all linux and bsd variants with minimal dependencies.

Note that this installation method is most recommended for ephemeral environments like CI jobs. Longer-lived environments that would like to receive updates via  package manager should install the CLI via that package manager.

```sh
(curl -Ls --tlsv1.2 --proto "=https" --retry 3 https://cli.doppler.com/install.sh || wget -t 3 -qO- https://cli.doppler.com/install.sh) | sh
```

You can find the source `install.sh` file in this repo's `scripts` directory.

## Docker

We currently publish a `dopplerhq/cli` Docker image based on `alpine`. For more info, check out our [Docker guide](https://docs.doppler.com/docs/docker-base-image-guide).

You can find all source Dockerfiles in this repo's `/docker` [folder](https://github.com/DopplerHQ/cli/tree/master/docker).

## GitHub Action

You can install the latest version of the CLI via GitHub Action. See the cli-action [repo](https://github.com/DopplerHQ/cli-action) for more info.

## Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb`, `.rpm`, and `.apk` packages. If a binary does not yet exist for the OS/architecture you use, please open a GitHub Issue.

# Verify Signature

You can verify the integrity and authenticity of any released artifact using Doppler's public GPG key. All release artifacts are signed and have a corresponding signature file. Release artifacts are available on the [Releases](https://github.com/DopplerHQ/cli/releases) page.

```sh
# fetch Doppler's signing key
gpg --keyserver keyserver.ubuntu.com --recv D3D593D50EE79DEC
# example: verify 'doppler_3.23.0_freebsd_amd64.tar.gz'
gpg --verify doppler_3.23.0_freebsd_amd64.tar.gz.sig doppler_3.23.0_freebsd_amd64.tar.gz || echo "Verification failed!"
```
