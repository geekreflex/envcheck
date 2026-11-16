# envcheck — development tasks
#
# Run `make help` for the available targets.

BINARY      := envcheck
PKG         := github.com/geekreflex/envcheck
BUILDINFO   := $(PKG)/internal/buildinfo
DIST        := dist

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

GO_LDFLAGS  := -s -w \
	-X $(BUILDINFO).version=$(VERSION) \
	-X $(BUILDINFO).commit=$(COMMIT) \
	-X $(BUILDINFO).date=$(DATE)

# Keep in sync with nothing else: CI reads this value straight out of this
# file, so the pinned linter version lives here and only here.
GOLANGCI_VERSION := v2.6.2

# Applied to release builds and to the Go tools fetched via `go run`. Neither
# needs cgo, and linking without it is faster and avoids depending on a
# working local C toolchain. Tests deliberately keep the default, because the
# race detector needs cgo on some platforms.
NO_CGO := CGO_ENABLED=0

.DEFAULT_GOAL := help

## help: list the available targets
.PHONY: help
help:
	@grep -hE '^## ' $(MAKEFILE_LIST) | sed -e 's/## //' | awk -F': ' '{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

## build: compile the binary into ./dist
.PHONY: build
build:
	@mkdir -p $(DIST)
	$(NO_CGO) go build -trimpath -ldflags '$(GO_LDFLAGS)' -o $(DIST)/$(BINARY) .
	@echo "built $(DIST)/$(BINARY) $(VERSION)"

## install: install the binary into GOBIN
.PHONY: install
install:
	$(NO_CGO) go install -trimpath -ldflags '$(GO_LDFLAGS)' .

## test: run the test suite with the race detector
.PHONY: test
test:
	go test -race -shuffle=on ./...

## cover: run tests and open the coverage report
.PHONY: cover
cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html
	@echo "report: coverage.html"

## bench: run benchmarks
.PHONY: bench
bench:
	go test -run '^$$' -bench=. -benchmem ./...

## fmt: format the code and tidy imports
.PHONY: fmt
fmt:
	go fmt ./...
	$(NO_CGO) go run golang.org/x/tools/cmd/goimports@latest -w -local $(PKG) .

## lint: run go vet and golangci-lint
.PHONY: lint
lint: vet
	$(NO_CGO) go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION) run

## vet: run go vet
.PHONY: vet
vet:
	go vet ./...

## tidy: tidy and verify module dependencies
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## check: everything CI runs, in one command
.PHONY: check
check: tidy vet lint test build

## snapshot: build a local release with GoReleaser, without publishing
.PHONY: snapshot
snapshot:
	$(NO_CGO) go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean

## clean: remove build and coverage artifacts
.PHONY: clean
clean:
	rm -rf $(DIST) coverage.out coverage.html
