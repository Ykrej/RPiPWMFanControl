#! /usr/bin/bash

set -e

GOOS=linux GOARCH=arm64 go build -o build/rpi-pwm-fancontrol
