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

## Linux

### Debian/Ubuntu (apt)

```sh
# add Bintray's GPG key
$ sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 379CE192D401AB61

# add Doppler's apt repo
$ echo "deb https://dl.bintray.com/dopplerhq/doppler-deb stable main" | sudo tee /etc/apt/sources.list.d/dopplerhq-doppler.list

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

## Shell script

You can bypass package managers and quickly install the latest version of the CLI via shell script. The script automatically downloads and installs the CLI binary most appropriate for your system's architecture. It is also fully POSIX compliant to support all linux and bsd variants with minimal dependencies.

Note that this installation method is only recommended for ephemeral environments like CI jobs. Longer-lived environments that expect to receive updates should install the CLI via a package manager.

```sh
(curl -Ls https://cli.doppler.com/install.sh || wget -qO- https://cli.doppler.com/install.sh) | sh
```

You can find the source `install.sh` file in this repo's `scripts` directory.

## Docker

We currently publish these Docker tags:
- `dopplerhq/cli` based on `alpine`
- `dopplerhq/cli:node` based on `node:lts-alpine`
- `dopplerhq/cli:python` based on `python:3-alpine`
- `dopplerhq/cli:ruby` based on `ruby:2-alpine`

You can find all source Dockerfiles in the `/docker` folder ([here](https://github.com/DopplerHQ/cli/tree/master/docker)).

## GitHub Action

You can install the latest version of the CLI via GitHub Action. See the cli-action [repo](https://github.com/DopplerHQ/cli-action) for more info.

### Example
Here's an example Dockerfile for a Node app:

```dockerfile
FROM dopplerhq/cli:3-node

# doppler args must be passed at runtime
ENV DOPPLER_TOKEN="" DOPPLER_PROJECT="" DOPPLER_CONFIG=""

COPY . .

# doppler will automatically use the environment variables specified above
ENTRYPOINT doppler run -- node index.js
```

Build the Dockerfile:

```sh
docker build -t mytestapp .
```

Then run the container:
```sh
docker run --rm -it -p 3000:3000 -e DOPPLER_TOKEN="" -e DOPPLER_PROJECT="" -e DOPPLER_CONFIG="" mytestapp
```

To avoid hard-coding the values, you can use the cli's `configure` command:

```sh
docker run --rm -it -p 3000:3000 -e DOPPLER_TOKEN="$(doppler configure get token --plain)" -e DOPPLER_PROJECT="$(doppler configure get project --plain)" -e DOPPLER_CONFIG="$(doppler configure get config --plain)" mytestapp
```

Flags:
- `--rm` delete the container once it exits
- `-i` attach to stdin; enables killing w/ ctrl+c
- `-t` print output to this terminal
- `-p 3000:3000` the port your app uses to service requests, if any
- `-e DOPPLER_TOKEN=""` pass a token into the environment
- `-e DOPPLER_PROJECT=""` pass an project into the environment
- `-e DOPPLER_CONFIG=""` pass an config into the environment

## Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb` and `.rpm` packages. If a binary does not yet exist for the OS/architecture you use, please open a GitHub Issue.

# Verify Signature

You can verify the integrity and authenticity of any released artifact using Doppler's public GPG key. The signatures of all release artifacts are placed in checksums.txt, which itself is then signed.

```sh
# fetch Doppler's signing key
gpg --keyserver keyserver.ubuntu.com --recv D3D593D50EE79DEC
# verify content of checksums.txt against signature
gpg --verify checksums.txt.sig checksums.txt
# verify checksum of cli binary (downloaded file name must match download page)
sha256sum --check --strict --ignore-missing checksums.txt
```

If the signature matches, you'll see output like this:
```sh
$ sha256sum --check --ignore-missing --strict checksums.txt
doppler_3.3.2_linux_amd64.deb: OK
```
