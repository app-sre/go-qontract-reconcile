#!/bin/bash
set -eu

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker --config="$DOCKER_CONF" login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

# lint code
make golint

# compile sources and run unit tests
make build

# verify if schema update causes issues
make update-schema validate
