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

Alternatively, you can download the doppler `pkg` file from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. This will install the doppler binary in `/usr/local/bin`. Note that this installation method does not support seamless updates. To update, you'll need to download and run the new `pkg` file.

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

## Docker

We currently build these Docker images:
- `dopplerhq/cli` based on `alpine`
- `dopplerhq/cli:node` based on `node:lts-alpine`
- `dopplerhq/cli:python` based on `python:3-alpine`
- `dopplerhq/cli:ruby` based on `ruby:2-alpine`

You can find all source Dockerfiles in the `/docker` folder.

Here's one such file:

```Dockerfile
FROM alpine
COPY doppler /bin/doppler
ENTRYPOINT ["/bin/doppler"]
```

To run an image, use `docker`:

```sh
# prints the doppler cli version
$ docker run --rm -it dopplerhq/cli:2-node --version
v2.0.0
# prints the node version
$ docker run --rm -it dopplerhq/cli:2-node run -- node --version
v12.13.0
```

Here's an example Dockerfile showing how you can build on top of Doppler's base images.

```dockerfile
FROM dopplerhq/cli:2-node

# doppler args are passed at runtime
ENV DOPPLER_TOKEN="" ENCLAVE_PROJECT="" ENCLAVE_CONFIG=""

COPY . .

# doppler will automatically use the DOPPLER_* and ENCLAVE_* environment variables
ENTRYPOINT doppler run -- node index.js
```

## Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb` and `.rpm` packages. If a binary does not yet exist for the OS/architecture you use, please open a GitHub Issue.
