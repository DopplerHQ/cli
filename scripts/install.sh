#!/usr/bin/env sh

set -e

# error codes
# 1 general
# 2 insufficient perms
# 3 gnupg package not installed
# 4 ~/.gnupg ownership issue

DOPPLER_DOMAIN="cli.doppler.com"
DEBUG=0
INSTALL=1
CLEAN_EXIT=0
USE_PACKAGE_MANAGER=1
DISABLE_CURL=0
CUSTOM_INSTALL_PATH=""
BINARY_INSTALLED_PATH=""

tempdir=""
filename=""
sig_filename=""
key_filename=""

cleanup() {
  exit_code=$?
  if [ "$exit_code" -ne 0 ] && [ "$CLEAN_EXIT" -ne 1 ]; then
    log "ERROR: script failed during execution"

    if [ "$DEBUG" -eq 0 ]; then
      log "For more verbose output, re-run this script with the debug flag (./install.sh --debug)"
    fi
  fi

  if [ -n "$tempdir" ]; then
    delete_tempdir
  fi

  clean_exit "$exit_code"
}
trap cleanup EXIT
trap cleanup INT

clean_exit() {
  CLEAN_EXIT=1
  exit "$1"
}

log() {
  # print to stderr
  >&2 echo "$1"
}

log_debug() {
  if [ "$DEBUG" -eq 1 ]; then
    # print to stderr
    >&2 echo "DEBUG: $1"
  fi
}

log_warning() {
  # print to stderr
  >&2 echo "WARNING: $1"
}

delete_tempdir() {
  log_debug "Removing temp directory"
  rm -rf "$tempdir"
  tempdir=""
}

linux_shell() {
  user="$(whoami)"
  grep -E "^$user:" < /etc/passwd | cut -f 7 -d ":" | head -1
}

macos_shell() {
  dscl . -read ~/ UserShell | sed 's/UserShell: //'
}

# we currently only support Git Bash for Windows with this script
# so the shell will always be /usr/bin/bash
windows_shell() {
  echo "/usr/bin/bash"
}

install_completions() {
  default_shell=""
  if [ "$os" = "macos" ]; then
    default_shell="$(macos_shell || true)"
  elif [ "$os" = "windows" ]; then
    default_shell="$(windows_shell || true)"
  else
    default_shell="$(linux_shell || true)"
  fi

  log_debug "Installing shell completions for '$default_shell'"
  # ignore all output
  "$BINARY_INSTALLED_PATH/"doppler completion install "$default_shell" --no-check-version > /dev/null 2>&1
}

# exit code
# 0=installed
# 1=path not writable
# 2=path not in PATH
# 3=path not a directory
# 4=path not found
install_binary() {
  install_dir="$1"
  # defaults to true
  require_dir_in_path="$2"
  # defaults to false
  create_if_not_exist="$3"

  if [ "$require_dir_in_path" != "false" ] && ! is_dir_in_path "$install_dir"; then
    return 2
  fi

  if [ "$create_if_not_exist" = "true" ] && [ ! -e "$install_dir" ]; then
    log_debug "$install_dir is in PATH but doesn't exist"
    log_debug "Creating $install_dir"
    mkdir -m 755 "$install_dir" > /dev/null 2>&1
  fi

  if [ ! -e "$install_dir" ]; then
    return 4
  fi

  if [ ! -d "$install_dir" ]; then
    return 3
  fi

  if ! is_path_writable "$install_dir"; then
    return 1
  fi

  log_debug "Moving binary to $install_dir"
  mv -f "$extract_dir/doppler" "$install_dir"
  return 0
}

curl_download() {
  url="$1"
  output_file="$2"
  component="$3"

  # allow curl to fail w/o exiting
  set +e
  headers=$(curl --tlsv1.2 --proto "=https" -w "%{http_code}" --silent --retry 5 -o "$output_file" -LN -D - "$url" 2>&1)
  exit_code=$?
  set -e

  status_code="$(echo "$headers" | tail -1)"

  if [ "$status_code" -ne 200 ]; then
    log_debug "Request failed with http status $status_code"
    log_debug "Response headers:"
    log_debug "$headers"
  fi

  if [ "$exit_code" -ne 0 ]; then
    log "ERROR: curl failed with exit code $exit_code"

    if [ "$exit_code" -eq 60 ]; then
      log ""
      log "Ensure the ca-certificates package is installed for your distribution"
    fi
    clean_exit 1
  fi

  if [ "$status_code" -eq 200 ]; then
    if [ "$component" = "Binary" ]; then
      parse_version_header "$headers"
    fi
  fi

  # this could be >255, so print HTTP status code rather than using as return code
  echo "$status_code"
}

# note: wget does not retry on 5xx
wget_download() {
  url="$1"
  output_file="$2"
  component="$3"

  security_flags="--secure-protocol=TLSv1_2 --https-only"
  # determine if using BusyBox wget (bad) or GNU wget (good)
  (wget --help 2>&1 | head -1 | grep -iv busybox > /dev/null 2>&1) || security_flags=""
  # only print this warning once per script invocation
  if [ -z "$security_flags" ] && [ "$component" = "Binary" ]; then
    log_debug "Skipping additional security flags that are unsupported by BusyBox wget"
    # log to stderr b/c this function's stdout is parsed
    log_warning "This system's wget binary is provided by BusyBox. Doppler strongly suggests installing GNU wget, which provides additional security features."
  fi

  # allow wget to fail w/o exiting
  set +e
  # we explicitly disable shellcheck here b/c security_flags isn't parsed properly when quoted
  # shellcheck disable=SC2086
  headers=$(wget $security_flags -q -t 5 -S -O "$output_file" "$url" 2>&1)
  exit_code=$?
  set -e

  status_code="$(echo "$headers" | grep -o -E '^\s*HTTP/[0-9.]+ [0-9]{3}' | tail -1 | grep -o -E '[0-9]{3}')"
  # it's possible for this value to be blank, so confirm that it's a valid status code
  valid_status_code=0
  if expr "$status_code" : '[0-9][0-9][0-9]$'>/dev/null; then
    valid_status_code=1
  fi

  if [ "$exit_code" -ne 0 ]; then
    if [ "$valid_status_code" -eq 1 ]; then
      # print the code and continue
      log_debug "Request failed with http status $status_code"
      log_debug "Response headers:"
      log_debug "$headers"
    else
      # exit immediately
      log "ERROR: wget failed with exit code $exit_code"

      if [ "$exit_code" -eq 5 ]; then
        log ""
        log "Ensure the ca-certificates package is installed for your distribution"
      fi
      clean_exit 1
    fi
  fi

  if [ "$status_code" -eq 200 ]; then
    if [ "$component" = "Binary" ]; then
      parse_version_header "$headers"
    fi
  fi

  # this could be >255, so print HTTP status code rather than using as return code
  echo "$status_code"
}

parse_version_header() {
  headers="$1"
  tag=$(echo "$headers" | sed -n 's/^[[:space:]]*x-cli-version: \(v[0-9]*\.[0-9]*\.[0-9]*\)[[:space:]]*$/\1/p')
  if [ -n "$tag" ]; then
    log_debug "Downloaded CLI $tag"
  fi
}

check_http_status() {
  status_code="$1"
  component="$2"

  if [ "$status_code" -ne 200 ]; then
    error="ERROR: $component download failed with status code $status_code."
    if [ "$status_code" -ne 404 ]; then
      error="${error} Please try again."
    fi

    log ""
    log "$error"

    if [ "$status_code" -eq 404 ]; then
      log ""
      log "Please report this issue:"
      log "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unexpected%20404%20using%20CLI%20install%20script"
    fi

    clean_exit 1
  fi
}

is_dir_in_path() {
  dir="$1"
  # ensure dir is the full path and not a substring of some longer path.
  # after performing a regex search, perform another search w/o regex to filter out matches due to special characters in `$dir`
  echo "$PATH" | grep -o -E "(^|:)$dir(:|$)" | grep -q -F "$dir"
}

is_path_writable() {
  dir="$1"
  test -w "$dir"
}

find_install_path_arg=0
# flag parsing
for arg; do
  if [ "$find_install_path_arg" -eq 1 ]; then
    CUSTOM_INSTALL_PATH="$arg"
    find_install_path_arg=0
    continue
  fi

  if [ "$arg" = "--debug" ]; then
    DEBUG=1
  fi

  if [ "$arg" = "--no-install" ]; then
    INSTALL=0
  fi

  if [ "$arg" = "--no-package-manager" ]; then
    USE_PACKAGE_MANAGER=0
  fi

  if [ "$arg" = "--disable-curl" ]; then
    DISABLE_CURL=1
  fi

  if [ "$arg" = "--install-path" ]; then
    find_install_path_arg=1
  fi
done

if [ "$find_install_path_arg" -eq 1 ]; then
  log "You must provide a path when specifying --install-path"
  clean_exit 1
fi

if [ "$CUSTOM_INSTALL_PATH" != "" ]; then
  # disable package managers when specifying custom path
  USE_PACKAGE_MANAGER=0
fi

# identify OS
os="unknown"
uname_os=$(uname -s)
case "$uname_os" in
  Darwin)    os="macos"   ;;
  Linux)     os="linux"   ;;
  FreeBSD)   os="freebsd" ;;
  OpenBSD)   os="openbsd" ;;
  NetBSD)    os="netbsd"  ;;
  *MINGW64*) os="windows" ;;
  *)
    log "ERROR: Unsupported OS '$uname_os'"
    log ""
    log "Please report this issue:"
    log "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unsupported%20OS"
    clean_exit 1
    ;;
esac

log_debug "Detected OS '$os'"

# disable package managers on macOS and windows (their use would be most unexpected)
if [ "$os" = "macos" ] || [ "$os" = "windows" ]; then
  USE_PACKAGE_MANAGER=0
fi

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
  log "ERROR: Unsupported architecture '$uname_machine'"
  log ""
  log "Please report this issue:"
  log "https://github.com/DopplerHQ/cli/issues/new?template=bug_report.md&title=[BUG]%20Unsupported%20architecture"
  clean_exit 1
fi

log_debug "Detected architecture '$arch'"

# identify format
if [ "$os" = "windows" ]; then
  format="zip"
else
  format="tar"
fi

if [ "$USE_PACKAGE_MANAGER" -eq 1 ]; then
  if [ -x "$(command -v dpkg)" ]; then
    format="deb"
  elif [ -x "$(command -v rpm)" ]; then
    format="rpm"
  fi
fi

log_debug "Detected format '$format'"

gpg_binary="$(command -v gpg || true)";
if [ -x "$gpg_binary" ]; then
  log_debug "Using $gpg_binary for signature verification"
else
  log "ERROR: Unable to find gpg binary for signature verification"
  log "You can resolve this error by installing your system's gnupg package"
  clean_exit 3
fi

url="https://$DOPPLER_DOMAIN/download?os=$os&arch=$arch&format=$format"
sig_url="https://$DOPPLER_DOMAIN/download/signature?os=$os&arch=$arch&format=$format"
key_url="https://$DOPPLER_DOMAIN/keys/public"


set +e
curl_binary="$(command -v curl)"
wget_binary="$(command -v wget)"

# check if curl is available
[ "$DISABLE_CURL" -eq 0 ] && [ -x "$curl_binary" ]
curl_installed=$? # 0 = yes

# check if wget is available
[ -x "$wget_binary" ]
wget_installed=$? # 0 = yes
set -e

if [ "$curl_installed" -eq 0 ] || [ "$wget_installed" -eq 0 ]; then
  # create hidden temp dir in user's home directory to ensure no other users have write perms
  tempdir="$(mktemp -d ~/.tmp.XXXXXXXX)"
  log_debug "Using temp directory $tempdir"

  log "Downloading Doppler CLI"
  file="doppler-download"
  filename="$tempdir/$file"
  sig_filename="$filename.sig"
  key_filename="$tempdir/publickey.gpg"

  if [ "$curl_installed" -eq 0 ]; then
    log_debug "Using $curl_binary for requests"

    # download binary
    log_debug "Downloading binary from $url"
    status_code=$( curl_download "$url" "$filename" "Binary" )
    check_http_status "$status_code" "Binary"

    # download signature
    log_debug "Downloading binary signature from $sig_url"
    status_code=$( curl_download "$sig_url" "$sig_filename" "Signature" )
    check_http_status "$status_code" "Signature"

    # download public key
    log_debug "Downloading public key from $key_url"
    status_code=$( curl_download "$key_url" "$key_filename" "Public key" )
    check_http_status "$status_code" "Public key"
  elif [ "$wget_installed" -eq 0 ]; then
    log_debug "Using $wget_binary for requests"

    log_debug "Downloading binary from $url"
    status_code=$( wget_download "$url" "$filename" "Binary" )
    check_http_status "$status_code" "Binary"

    # download signature
    log_debug "Download binary signature from $sig_url"
    status_code=$( wget_download "$sig_url" "$sig_filename" "Signature" )
    check_http_status "$status_code" "Signature"

    # download public key
    log_debug "Download public key from $key_url"
    status_code=$( wget_download "$key_url" "$key_filename" "Public key" )
    check_http_status "$status_code" "Public key"
  fi
else
  log "ERROR: You must have curl or wget installed"
  clean_exit 1
fi

log "Verifying signature"
# verify we can read ~/.gnupg so that we can provide a helpful error message
if [ -d ~/.gnupg ]; then
  # Run sudo chown -r $(whoami) ~/.gnupg to fix this
  ls -l ~/.gnupg > /dev/null 2>&1 || (log "Failed to read ~/.gnupg. Please verify the directory's ownership, or run 'sudo chown -R $(whoami) ~/.gnupg' to fix this." && clean_exit 4)
fi
gpg --no-default-keyring --keyring "$key_filename" --verify "$sig_filename" "$filename" > /dev/null 2>&1 || (log "Failed to verify binary signature" && clean_exit 1)
log_debug "Signature successfully verified!"

if [ "$format" = "deb" ]; then
  mv -f "$filename" "$filename.deb"
  filename="$filename.deb"

  if [ "$INSTALL" -eq 1 ]; then
    log "Installing..."
    dpkg -i "$filename"
    # dpkg doesn't provide us the actual path, so take a best guess
    BINARY_INSTALLED_PATH="$(dirname "$(command -v doppler)")"
    echo "Installed Doppler CLI $("$BINARY_INSTALLED_PATH/"doppler -v)"
  else
    log_debug "Moving installer to $(pwd) (cwd)"
    mv -f "$filename" .
    echo "Doppler CLI installer saved to ./$file.deb"
  fi
elif [ "$format" = "rpm" ]; then
  mv -f "$filename" "$filename.rpm"
  filename="$filename.rpm"

  if [ "$INSTALL" -eq 1 ]; then
    log "Installing..."
    rpm -i --force "$filename"
    # rpm doesn't provide us the actual path, so take a best guess
    BINARY_INSTALLED_PATH="$(dirname "$(command -v doppler)")"
    echo "Installed Doppler CLI $("$BINARY_INSTALLED_PATH/"doppler -v)"
  else
    log_debug "Moving installer to $(pwd) (cwd)"
    mv -f "$filename" .
    echo "Doppler CLI installer saved to ./$file.rpm"
  fi
elif [ "$format" = "tar" ] || [ "$format" = "zip" ]; then
  if [ "$format" = "tar" ]; then
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
  elif [ "$format" = "zip" ]; then
    mv -f "$filename" "$filename.zip"
    filename="$filename.zip"

    # extract
    extract_dir="$tempdir/x"
    mkdir "$extract_dir"
    log_debug "Extracting zip to $extract_dir"
    unzip -d "$extract_dir" "$filename"

    # set appropriate perms
    chown "$(id -u):$(id -g)" "$extract_dir/doppler"
    chmod 755 "$extract_dir/doppler"
  fi

  # install
  if [ "$INSTALL" -eq 1 ]; then
    log "Installing..."
    binary_installed=0
    found_non_writable_path=0

    if [ "$CUSTOM_INSTALL_PATH" != "" ]; then
      # install to this directory or fail; don't try any other paths

      # capture exit code without exiting
      set +e
      install_binary "$CUSTOM_INSTALL_PATH" "false" "false"
      exit_code=$?
      set -e
      if [ $exit_code -eq 0 ]; then
        binary_installed=1
        BINARY_INSTALLED_PATH="$CUSTOM_INSTALL_PATH"
      elif [ $exit_code -eq 1 ]; then
        log "Install path is not writable: \"$CUSTOM_INSTALL_PATH\""
        clean_exit 2
      elif [ $exit_code -eq 4 ]; then
        log "Install path does not exist: \"$CUSTOM_INSTALL_PATH\""
        clean_exit 1
      else
        log "Install path is not a valid directory: \"$CUSTOM_INSTALL_PATH\""
        clean_exit 1
      fi
    fi

    # check for an existing Doppler binary
    if [ "$binary_installed" -eq 0 ]; then
      existing_install_dir="$(command -v doppler || true)"
      if [ "$existing_install_dir" != "" ]; then
        install_dir="$(dirname "$existing_install_dir")"
        # capture exit code without exiting
        set +e
        install_binary "$install_dir"
        exit_code=$?
        set -e
        if [ $exit_code -eq 0 ]; then
          binary_installed=1
          BINARY_INSTALLED_PATH="$install_dir"
        elif [ $exit_code -eq 1 ]; then
          found_non_writable_path=1
        fi
      fi
    fi

    if [ "$binary_installed" -eq 0 ]; then
      install_dir="/usr/local/bin"
      # capture exit code without exiting
      set +e
      install_binary "$install_dir"
      exit_code=$?
      set -e
      if [ $exit_code -eq 0 ]; then
        binary_installed=1
        BINARY_INSTALLED_PATH="$install_dir"
      elif [ $exit_code -eq 1 ]; then
        found_non_writable_path=1
      fi
    fi

    if [ "$binary_installed" -eq 0 ]; then
      install_dir="/usr/bin"
      # capture exit code without exiting
      set +e
      install_binary "$install_dir"
      exit_code=$?
      set -e
      if [ $exit_code -eq 0 ]; then
        binary_installed=1
        BINARY_INSTALLED_PATH="$install_dir"
      elif [ $exit_code -eq 1 ]; then
        found_non_writable_path=1
      fi
    fi

    if [ "$binary_installed" -eq 0 ]; then
      install_dir="/usr/sbin"
      # capture exit code without exiting
      set +e
      install_binary "$install_dir"
      exit_code=$?
      set -e
      if [ $exit_code -eq 0 ]; then
        binary_installed=1
        BINARY_INSTALLED_PATH="$install_dir"
      elif [ $exit_code -eq 1 ]; then
        found_non_writable_path=1
      fi
    fi

    if [ "$binary_installed" -eq 0 ]; then
      # run again for this directory, but this time create it if it doesn't exist
      # this fixes an issue with clean installs on macOS 12+
      install_dir="/usr/local/bin"
      # capture exit code without exiting
      set +e
      install_binary "$install_dir" "true" "true"
      exit_code=$?
      set -e
      if [ $exit_code -eq 0 ]; then
        binary_installed=1
        BINARY_INSTALLED_PATH="$install_dir"
      elif [ $exit_code -eq 1 ]; then
        found_non_writable_path=1
      fi
    fi

    if [ "$binary_installed" -eq 0 ]; then
      if [ "$found_non_writable_path" -eq 1 ]; then
        log "Unable to write to bin directory; please re-run with \`sudo\` or adjust your PATH"
        clean_exit 2
      else
        log "No supported bin directories are available; please adjust your PATH"
        clean_exit 1
      fi
    fi
  else
    log_debug "Moving binary to $(pwd) (cwd)"
    mv -f "$extract_dir/doppler" .
  fi

  delete_tempdir

  if [ "$INSTALL" -eq 1 ]; then
    message="Installed Doppler CLI $("$BINARY_INSTALLED_PATH"/doppler -v)"
    if [ "$CUSTOM_INSTALL_PATH" != "" ]; then
      message="$message to $BINARY_INSTALLED_PATH"
    fi
    echo "$message"
  else
    echo "Doppler CLI saved to ./doppler"
  fi
fi

install_completions || log_debug "Unable to install shell completions"
