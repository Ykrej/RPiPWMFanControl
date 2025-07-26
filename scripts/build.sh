#! /usr/bin/bash

set -e

export GOARCH=arm64  # Default arm64 because rpi target
export GOOS=linux
export CGO_ENABLED=0

while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      VERSION="$2"
      shift # past argument
      shift # past value
      ;;
    --go-arch)
      GOARCH="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done


set -ex

if [[ -n "$VERSION" ]]; then
  LDFLAGS="-ldflags=-X main.Version=$VERSION"
  go build "$LDFLAGS" -v -o build/rpi-pwm-fancontrol
  exit 0
else
  go build -v -o build/rpi-pwm-fancontrol
fi
