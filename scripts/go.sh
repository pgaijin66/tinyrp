#!/bin/bash

set -euo pipefail

BIN="tinyrp"

# remove old bin directory and recreate it
rm -f bin/*
mkdir -p bin/

OS="$(go env GOOS)"
ARCH="$(go env GOARCH)"

GIT_COMMIT_SHA=$(git rev-parse HEAD)
BUILD_DATE=$(date -u '+%Y-%m-%d %H:%M:%S')


if [ "$#" -ne 1 ];
then
    echo "[-] Error: No positional arguments provided. Specify 'build' or 'run'"
    exit 1

elif [ "$1" == "build" ];
then
    echo "[*] Building go binary"
    GOOS=${OS} GOARCH=${ARCH} go build -ldflags "-X config.Version=1.0.0" -o "/usr/local/bin/${BIN}" cmd/main.go

elif [ "$1" == "run" ];
then
    echo "[*] Running go binary"
    GOOS=${OS} GOARCH=${ARCH} go run -ldflags "${LD_FLAGS}" cmd/main.go

fi