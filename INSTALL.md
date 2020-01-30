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

We currently publish these Docker tags:
- `dopplerhq/cli` based on `alpine`
- `dopplerhq/cli:node` based on `node:lts-alpine`
- `dopplerhq/cli:python` based on `python:3-alpine`
- `dopplerhq/cli:ruby` based on `ruby:2-alpine`

You can find all source Dockerfiles in the `/docker` folder ([here](https://github.com/DopplerHQ/cli/tree/master/docker)).

### Example
Here's an example Dockerfile for a Node app:

```dockerfile
FROM dopplerhq/cli:2-node

# doppler args must be passed at runtime
ENV DOPPLER_TOKEN="" ENCLAVE_PROJECT="" ENCLAVE_CONFIG=""

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
docker run --rm -it -p 3000:3000 -e DOPPLER_TOKEN="" -e ENCLAVE_PROJECT="" -e ENCLAVE_CONFIG="" mytestapp
```

To avoid hard-coding the values, you can use the cli's `configure` command:

```sh
docker run --rm -it -p 3000:3000 -e DOPPLER_TOKEN="$(doppler configure get token --plain)" -e ENCLAVE_PROJECT="$(doppler configure get enclave.project --plain)" -e ENCLAVE_CONFIG="$(doppler configure get enclave.config --plain)" mytestapp
```

Flags:
- `--rm` delete the container once it exits
- `-i` attach to stdin; enables killing w/ ctrl+c
- `-t` print output to this terminal
- `-p 3000:3000` the port your app uses to service requests, if any
- `-e DOPPLER_TOKEN=""` pass a token into the environment
- `-e ENCLAVE_PROJECT=""` pass an enclave project into the environment
- `-e ENCLAVE_CONFIG=""` pass an enclave config into the environment

## Other

You can download all binaries and release artifacts from the [Releases](https://github.com/DopplerHQ/cli/releases/latest) page. Binaries are built for macOS, Linux, Windows, FreeBSD, OpenBSD, and NetBSD, and for 32-bit, 64-bit, armv6/armv7, and armv6/armv7 64-bit architectures.

You can also directly download the generated `.deb` and `.rpm` packages. If a binary does not yet exist for the OS/architecture you use, please open a GitHub Issue.
