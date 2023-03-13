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
make update-schema validate-schema

# We must use the same version as the Terraform provider.
grep -q 'github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7' go.mod || {
    echo "go-crypto package version mismatch!"
    exit 1
} >&2
