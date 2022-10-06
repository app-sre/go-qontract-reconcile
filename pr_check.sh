#!/bin/bash
set -eu

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker --config="$DOCKER_CONF" login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

# compile sources and run unit tests
make build

# verify if schema update causes issues
make update-schema validate-schema

# We must use the same version as the terraform provider here.
grep -q 'github.com/keybase/go-crypto v0.0.0-20161004153544-93f5b35093ba' go.mod || (echo "go-crypto version mismatch"; exit 1)

