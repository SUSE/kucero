MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
.SUFFIXES:

# The semver version number which will be used as the Docker image tag
# Defaults to the output of git describe.
VERSION ?= $(shell git describe --tags --dirty --always)

# Docker image name parameters
DOCKER_NAME ?= jenting/kucero
DOCKER_TAG ?= ${VERSION}
DOCKER_IMAGE ?= ${DOCKER_NAME}:${DOCKER_TAG}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

BIN := ${CURDIR}/bin
export PATH := ${BIN}:${PATH}

all: kucero

verify:
	go mod tidy
	go mod verify

test: verify
	go test -count=1 ./...

kucero: test
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o cmd/kucero/kucero cmd/kucero/*.go

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: test
	docker build . -t ${DOCKER_IMAGE}

# Push the docker image
docker-push:
	docker push ${DOCKER_IMAGE}

# Deploy manifest
deploy-manifest:
	cd manifest && kustomize edit set image kucero=${DOCKER_IMAGE}
	kustomize build manifest | kubectl apply -f -

# Destroy manifest
destroy-manifest:
	kustomize build manifest | kubectl delete -f -

clean:
	go clean -x -i ./...
