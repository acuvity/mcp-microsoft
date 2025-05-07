MAKEFLAGS += --warn-undefined-variables
SHELL := /bin/bash -o pipefail

CONTAINER_ENGINE ?= docker
CONTAINER_TAG ?= "latest"
OCI_REGISTRY ?= "acuvity"

export GO111MODULE = on

default: lint test build vuln sec

lint:
	golangci-lint run \
		--timeout=5m \
		--disable=govet  \
		--enable=errcheck \
		--enable=ineffassign \
		--enable=unused \
		--enable=unconvert \
		--enable=misspell \
		--enable=prealloc \
		--enable=nakedret \
		--enable=unparam \
		--enable=nilerr \
		--enable=bodyclose \
		--enable=errorlint \
		./...
test:
	go test ./... -vet off -race -cover -covermode=atomic -coverprofile=unit_coverage.out

sec:
	gosec -quiet ./...

vuln:
	govulncheck ./...

build:
	env GOOS=linux GOARCH=amd64 go build .

remod:
	go mod tidy

mcp-server: build
	@$(CONTAINER_ENGINE) buildx build --attest type=sbom --attest type=provenance --platform linux/arm64/v8,linux/amd64 --tag $(OCI_REGISTRY)/mcp-server-microsoft-graph:$(CONTAINER_TAG) .

mcp-server-push: build
	@$(CONTAINER_ENGINE) buildx build --attest type=sbom --attest type=provenance --push --platform linux/arm64/v8,linux/amd64 --tag $(OCI_REGISTRY)/mcp-server-microsoft-graph:$(CONTAINER_TAG) .
