.PHONY: build push gotest gobuild govet coveragereport

CONTAINER_ENGINE ?= $(shell which podman >/dev/null 2>&1 && echo podman || echo docker)

IMAGE_NAME := quay.io/app-sre/user-validator
IMAGE_TAG := $(shell git rev-parse --short=7 HEAD)

ifneq (,$(wildcard $(CURDIR)/.docker))
	DOCKER_CONF := $(CURDIR)/.docker
else
	DOCKER_CONF := $(HOME)/.docker
endif

GOOS := $(shell go env GOOS)
TMP_COVERAGE := $(shell mktemp)

gotest:
	CGO_ENABLED=0 GOOS=$(GOOS) go test ./...

govet: gotest
	go vet ./...

gobuild: govet
	CGO_ENABLED=0 GOOS=$(GOOS) go build -o user-validator -a ./main.go

build:
ifeq ($(CONTAINER_ENGINE), podman)
	@DOCKER_BUILDKIT=1 $(CONTAINER_ENGINE) build --no-cache -t $(IMAGE_NAME):latest . --progress=plain
else
	@DOCKER_BUILDKIT=1 $(CONTAINER_ENGINE) --config=$(DOCKER_CONF) build --no-cache -t $(IMAGE_NAME):latest . --progress=plain
endif
	@$(CONTAINER_ENGINE) tag $(IMAGE_NAME):latest $(IMAGE_NAME):$(IMAGE_TAG)

push:
	@$(CONTAINER_ENGINE) --config=$(DOCKER_CONF) push $(IMAGE_NAME):latest
	@$(CONTAINER_ENGINE) --config=$(DOCKER_CONF) push $(IMAGE_NAME):$(IMAGE_TAG)

coveragereport:
	go test -coverprofile=$(TMP_COVERAGE) ./...
	go tool cover -html=$(TMP_COVERAGE) -o coverage.html