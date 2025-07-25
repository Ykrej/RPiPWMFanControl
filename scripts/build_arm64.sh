#! /usr/bin/bash

GOOS=linux GOARCH=arm64 go build -o build/rpi-pwm-fancontrol
