#!/usr/bin/env sh

set -e

DEBUG=0
INSTALL=1
CLEAN_EXIT=0
USE_PACKAGE_MANAGER=1
tempdir=""
filename=""

cleanup() {
  exit_code=$?
  if [ "$exit_code" -ne 0 ] && [ "$CLEAN_EXIT" -ne 1 ]; then
    echo "ERROR: script failed during execution"

    if [ "$DEBUG" -eq 0 ]; then
      echo "For more verbose output, re-run this script with the debug flag (./install.sh --debug)"
    fi
  fi

  if [ -n "$tempdir" ]; then
    delete_tempdir
  fi

  exit "$exit_code"
}
trap cleanup EXIT

clean_exit() {
  CLEAN_EXIT=1
  exit "$1"
}

log_debug() {
  if [ "$DEBUG" -eq 1 ]; then
    echo "DEBUG: $1"
  fi
}

delete_tempdir() {
  log_debug "Removing temp directory"
  rm -rf "$tempdir"
  tempdir=""
}

# flag parsing
for arg; do
  if [ "$arg" = "--debug" ]; then
    DEBUG=1
  fi

  if [ "$arg" = "--no-install" ]; then
    INSTALL=0
  fi

  if [ "$arg" = "--no-package-manager" ]; then
    USE_PACKAGE_MANAGER=0
  fi
done

# identify OS
os="unknown"
uname_os=$(uname -s)
if [ "$uname_os" = "Darwin" ]; then
  os="macos"
elif [ "$uname_os" = "Linux" ]; then
  os="linux"
elif [ "$uname_os" = "FreeBSD" ]; then
  os="freebsd"
elif [ "$uname_os" = "OpenBSD" ]; then
  os="openbsd"
elif [ "$uname_os" = "NetBSD" ]; then
  os="netbsd"
else
  echo "ERROR: Unsupported OS '$uname_os'"
  echo ""
  echo "Please report this issue:"
  echo "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unsupported%20OS"
  clean_exit 1
fi

log_debug "Detected OS '$os'"

# identify arch
arch="unknown"
uname_machine=$(uname -m)
if [ "$uname_machine" = "i386" ] || [ "$uname_machine" = "i686" ]; then
  arch="i386"
elif [ "$uname_machine" = "amd64" ] || [ "$uname_machine" = "x86_64" ]; then
  arch="amd64"
elif [ "$uname_machine" = "armv6" ] || [ "$uname_machine" = "armv6l" ]; then
  arch="armv6"
elif [ "$uname_machine" = "armv7" ] || [ "$uname_machine" = "armv7l" ]; then
  arch="armv7"
# armv8?
elif [ "$uname_machine" = "arm64" ] || [ "$uname_machine" = "aarch64" ]; then
  arch="arm64"
else
  echo "ERROR: Unsupported architecture '$uname_machine'"
  echo ""
  echo "Please report this issue:"
  echo "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unsupported%20architecture"
  clean_exit 1
fi

log_debug "Detected architecture '$arch'"

# identify format
format="tar"
if [ "$USE_PACKAGE_MANAGER" -eq 1 ]; then
  if [ -x "$(command -v dpkg)" ]; then
    format="deb"
  elif [ -x "$(command -v rpm)" ]; then
    format="rpm"
  fi
fi

log_debug "Detected format '$format'"

url="https://cli.doppler.com/download?os=$os&arch=$arch&format=$format"

# download binary
if [ -x "$(command -v curl)" ] || [ -x "$(command -v wget)" ]; then
  # create hidden temp dir in user's home directory to ensure no other users have write perms
  tempdir="$(mktemp -d ~/.tmp.XXXXXXXX)"
  log_debug "Using temp directory $tempdir"

  echo "Downloading latest release"
  file="doppler-download"
  filename="$tempdir/$file"

  if [ -x "$(command -v curl)" ]; then
    log_debug "Using $(command -v curl)"
    log_debug "Downloading from $url"
    # when this fails print the exit code
    headers=$(curl --silent --retry 3 -o "$filename" -LN -D - "$url" || echo "$?")
    if expr "$headers" : '[0-9][0-9]*$'>/dev/null; then
      exit_code="$headers"
      echo "ERROR: curl failed with exit code $exit_code"

      if [ "$exit_code" -eq 60 ]; then
        echo ""
        echo "Ensure that CA Certificates are installed for your distribution"
      fi
      clean_exit 1
    fi
  else
    log_debug "Using $(command -v wget)"
    log_debug "Downloading from $url"
    # when this fails print the exit code
    headers=$(wget -q -t 3 -S -O "$filename" "$url" 2>&1 || echo "$?")
    if expr "$headers" : '[0-9][0-9]*$'>/dev/null; then
      exit_code="$headers"
      echo "ERROR: wget failed with exit code $exit_code"

      if [ "$exit_code" -eq 5 ]; then
        echo ""
        echo "Ensure that CA Certificates are installed for your distribution"
      fi
      clean_exit 1
    fi
  fi

  status=$(echo "$headers" | head -1 | sed -n 's/^[[:space:]]*HTTP.* \([0-9][0-9][0-9]\).*$/\1/p')
  if [ "$status" -ne 302 ]; then
    echo "ERROR: Download failed with status $status"

    if [ "$status" -eq 404 ]; then
      echo ""
      echo "Please report this issue:"
      echo "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unexpected%20404%20using%20CLI%20install%20script"
    fi

    clean_exit 1
  fi
else
  echo "ERROR: You must have curl or wget installed"
  clean_exit 1
fi

tag=$(echo "$headers" | sed -n 's/^[[:space:]]*x-cli-version: \(v[0-9]*\.[0-9]*\.[0-9]*\)[[:space:]]*$/\1/p')
log_debug "Downloaded CLI $tag"

if [ "$format" = "deb" ]; then
  mv -f "$filename" "$filename.deb"
  filename="$filename.deb"

  if [ "$INSTALL" -eq 1 ]; then
    echo 'Installing...'
    dpkg -i "$filename"
    echo "Installed Doppler CLI $(doppler -v)"
  else
    log_debug "Moving installer to $(pwd) (cwd)"
    mv -f "$filename" .
    echo "Doppler CLI installer saved to ./$file.deb"
  fi
elif [ "$format" = "rpm" ]; then
  mv -f "$filename" "$filename.rpm"
  filename="$filename.rpm"

  if [ "$INSTALL" -eq 1 ]; then
    echo 'Installing...'
    rpm -i --force "$filename"
    echo "Installed Doppler CLI $(doppler -v)"
  else
    log_debug "Moving installer to $(pwd) (cwd)"
    mv -f "$filename" .
    echo "Doppler CLI installer saved to ./$file.rpm"
  fi
elif [ "$format" = "tar" ]; then
  mv -f "$filename" "$filename.tar.gz"
  filename="$filename.tar.gz"

  # extract
  extract_dir="$tempdir/x"
  mkdir "$extract_dir"
  log_debug "Extracting tarball to $extract_dir"
  tar -xzf "$filename" -C "$extract_dir"

  # set appropriate perms
  chown "$(id -u):$(id -g)" "$extract_dir/doppler"
  chmod 755 "$extract_dir/doppler"

  # install
  if [ "$INSTALL" -eq 1 ]; then
    echo 'Installing...'
    log_debug "Moving binary to /usr/local/bin"
    mv -f "$extract_dir/doppler" /usr/local/bin
    if [ ! -x "$(command -v doppler)" ]; then
      log_debug "Binary not in PATH, moving to /usr/bin"
      mv -f /usr/local/bin/doppler /usr/bin/doppler
    fi
  else
    log_debug "Moving binary to $(pwd) (cwd)"
    mv -f "$extract_dir/doppler" .
  fi

  delete_tempdir

  if [ "$INSTALL" -eq 1 ]; then
    echo "Installed Doppler CLI $(doppler -v)"
  else
    echo "Doppler CLI saved to ./doppler"
  fi
fi
