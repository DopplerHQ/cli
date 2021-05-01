#!/usr/bin/env sh

set -e

DEBUG=0
INSTALL=1
CLEAN_EXIT=0
USE_PACKAGE_MANAGER=1
VERIFY_SIGNATURE=1
FORCE_VERIFY_SIGNATURE=0

tempdir=""
filename=""
sig_filename=""
key_filename=""

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
trap cleanup INT

clean_exit() {
  CLEAN_EXIT=1
  exit "$1"
}

log_debug() {
  if [ "$DEBUG" -eq 1 ]; then
    # print to stderr
    >&2 echo "DEBUG: $1"
  fi
}

delete_tempdir() {
  log_debug "Removing temp directory"
  rm -rf "$tempdir"
  tempdir=""
}

linux_shell() {
  user="$(whoami)"
  grep "$user" < /etc/passwd | cut -f 7 -d ":" | head -1
}

macos_shell() {
  dscl . -read ~/ UserShell | sed 's/UserShell: //'
}

install_completions() {
  default_shell=""
  if [ "$os" = "macos" ]; then
    default_shell="$(macos_shell || "")"
  else
    default_shell="$(linux_shell || "")"
  fi

  log_debug "Installing shell completions for '$default_shell'"
  # ignore all output
  doppler completion install "$default_shell" --no-check-version > /dev/null 2>&1
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

  if [ "$arg" = "--no-verify-signature" ]; then
    VERIFY_SIGNATURE=0
    echo "Disabling signature verification, this is not recommended"
  fi

  if [ "$arg" = "--verify-signature" ]; then
    VERIFY_SIGNATURE=1
    FORCE_VERIFY_SIGNATURE=1
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
sig_url="https://cli.doppler.com/download/signature?os=$os&arch=$arch&format=$format"
key_url="https://cli.doppler.com/keys/public"

if [ "$VERIFY_SIGNATURE" -eq 1 ]; then
  log_debug "Checking for gpg binary"
  if [ ! -x "$(command -v gpg)" ]; then
    if [ "$FORCE_VERIFY_SIGNATURE" -eq 1 ]; then
      echo "ERROR: Unable to find gpg binary for signature verficiation"
      echo "You can resolve this error by installing your system's gnupg package"
      clean_exit 1
    else
      log_debug "Unable to find gpg binary, skipping signature verification"
      VERIFY_SIGNATURE=0
      echo "WARNING: Skipping signature verification due to no available gpg binary"
      echo "Signature verification is an additional measure to ensure you're executing code that Doppler produced"
      echo "You can remove this warning by installing your system's gnupg package, or by specifying --no-verify-signature"
    fi
  fi
fi

# download binary
if [ -x "$(command -v curl)" ] || [ -x "$(command -v wget)" ]; then
  # create hidden temp dir in user's home directory to ensure no other users have write perms
  tempdir="$(mktemp -d ~/.tmp.XXXXXXXX)"
  log_debug "Using temp directory $tempdir"

  echo "Downloading Doppler CLI"
  file="doppler-download"
  filename="$tempdir/$file"
  sig_filename="$filename.sig"
  key_filename="$tempdir/publickey.gpg"

  if [ -x "$(command -v curl)" ]; then
    log_debug "Using $(command -v curl) for requests"
    log_debug "Downloading binary from $url"
    # when this fails print the exit code
    headers=$(curl --tlsv1.2 --proto "=https" --silent --retry 3 -o "$filename" -LN -D - "$url" || echo "$?")
    if expr "$headers" : '[0-9][0-9]*$'>/dev/null; then
      exit_code="$headers"
      echo "ERROR: curl failed with exit code $exit_code"

      if [ "$exit_code" -eq 60 ]; then
        echo ""
        echo "Ensure that CA Certificates are installed for your distribution"
      fi
      clean_exit 1
    fi

    if [ "$VERIFY_SIGNATURE" -eq 1 ]; then
      # download signature
      log_debug "Download binary signature from $sig_url"
      curl --fail --tlsv1.2 --proto "=https" --silent --retry 3 -o "$sig_filename" -LN "$sig_url" > /dev/null 2>&1 || (echo "Failed to download signature" && clean_exit 1)

      # download public key
      log_debug "Download public key from $key_url"
      curl --fail --tlsv1.2 --proto "=https" --silent --retry 3 -o "$key_filename" -LN "$key_url" > /dev/null 2>&1 || (echo "Failed to download public key" && clean_exit 1)
    fi
  else
    log_debug "Using $(command -v wget) for requests"

    # determine what features are supported by this version of wget (BusyBox wget is limited)
    security_flags="--secure-protocol=TLSv1_2 --https-only"
    (wget --help 2>&1 | head -1 | grep -iv busybox > /dev/null 2>&1) || security_flags=""
    if [ -z "$security_flags" ]; then
      log_debug "Skipping additional security flags that are unsupported by BusyBox wget"
    fi

    log_debug "Downloading binary from $url"

    # when this fails print the exit code
    # we explicitly disable shellcheck here b/c security_flags isn't parsed properly when quoted
    # shellcheck disable=SC2086
    headers=$(wget $security_flags -q -t 3 -S -O "$filename" "$url" 2>&1 || echo "$?")
    if expr "$headers" : '[0-9][0-9]*$'>/dev/null; then
      exit_code="$headers"
      echo "ERROR: wget failed with exit code $exit_code"

      if [ "$exit_code" -eq 5 ]; then
        echo ""
        echo "Ensure that CA Certificates are installed for your distribution"
      fi
      clean_exit 1
    fi

    if [ "$VERIFY_SIGNATURE" -eq 1 ]; then
      # download signature
      log_debug "Download binary signature from $sig_url"
      # we explicitly disable shellcheck here b/c security_flags isn't parsed properly when quoted
      # shellcheck disable=SC2086
      wget $security_flags -q -t 3 -S -O "$sig_filename" "$sig_url" > /dev/null 2>&1 || (echo "Failed to download signature" && clean_exit 1)

      # download public key
      log_debug "Download public key from $key_url"
      # we explicitly disable shellcheck here b/c security_flags isn't parsed properly when quoted
      # shellcheck disable=SC2086
      wget $security_flags -q -t 3 -S -O "$key_filename" "$key_url" > /dev/null 2>&1 || (echo "Failed to download public key" && clean_exit 1)
    fi
  fi

  status=$(echo "$headers" | head -1 | sed -n 's/^[[:space:]]*HTTP.* \([0-9][0-9][0-9]\).*$/\1/p')
  if [ "$status" -ne 302 ]; then
    echo "ERROR: Download failed with status $status"

    log_debug "Response headers:"
    log_debug "$headers"

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

if [ "$VERIFY_SIGNATURE" -eq 1 ]; then
  log_debug "Verifying GPG signature"
  gpg --no-default-keyring --keyring "$key_filename" --verify "$sig_filename" "$filename" > /dev/null 2>&1 || (echo "Failed to verify binary signature" && clean_exit 1)
  log_debug "Signature successfully verified!"
else
  log_debug "Skipping signature verification"
fi

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

install_completions || log_debug "Unable to install shell completions"
