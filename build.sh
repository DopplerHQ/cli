#!/bin/sh

targets=(
  "darwin-amd64"
  "freebsd-amd64"
  "freebsd-arm"
  "linux-386"
  "linux-amd64"
  "openbsd-386"
  "openbsd-amd64"
  "windows-386-exe"
  "windows-amd64-exe"
)

IFS='-'
for target in "${targets[@]}"; do
  read -ra TARGET <<< "$target"

  os="${TARGET[0]}"
  arch="${TARGET[1]}"

  echo "Building binary for $os $arch"

  if [ "${#TARGET[@]}" == 2 ]; then
    GOOS="$os" GOARCH="$arcg" go build -o "./bin/doppler-run.$os-$arch"
  fi

  extension="${TARGET[2]}"

  if [ "${#TARGET[@]}" == 3 ]; then
    GOOS="$os" GOARCH="$arcg" go build -o "./bin/doppler-run.$os-$arch.$extension"
  fi
done
