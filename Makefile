BINARY := htpasswd
CMD := ./cmd/htpasswd
BUILD_DIR := bin
DIST_DIR := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

PACKAGE := $(BINARY)_$(VERSION)_$(GOOS)_$(GOARCH)
PACKAGE_DIR := $(DIST_DIR)/$(PACKAGE)
ifeq ($(GOOS),windows)
BINARY_NAME := $(BINARY).exe
ARCHIVE := $(PACKAGE).zip
else
BINARY_NAME := $(BINARY)
ARCHIVE := $(PACKAGE).tar.gz
endif

.DEFAULT_GOAL := build

.PHONY: all build install test test-race fmt fmt-check vet lint check package release checksums clean help

all: check build

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD)

install:
	go install $(CMD)

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))" || \
		{ echo "Go files need formatting; run 'make fmt'"; gofmt -l $$(find . -name '*.go' -not -path './vendor/*'); exit 1; }

vet:
	go vet ./...

lint: fmt-check vet

check: lint test

# Build one release archive. Override VERSION, GOOS, and GOARCH as needed.
package:
	@rm -rf "$(PACKAGE_DIR)"
	@mkdir -p "$(PACKAGE_DIR)"
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		go build -trimpath -ldflags="-s -w" -o "$(PACKAGE_DIR)/$(BINARY_NAME)" $(CMD)
	@cp LICENSE README.md "$(PACKAGE_DIR)/"
ifeq ($(GOOS),windows)
	@cd $(DIST_DIR) && zip -Xqr "$(ARCHIVE)" "$(PACKAGE)"
else
	@COPYFILE_DISABLE=1 tar --no-xattrs --no-acls -C $(DIST_DIR) -czf "$(DIST_DIR)/$(ARCHIVE)" "$(PACKAGE)"
endif
	@rm -rf "$(PACKAGE_DIR)"
	@echo "Created $(DIST_DIR)/$(ARCHIVE)"

# Build all archives published by GitHub Actions and generate their checksums.
release: clean
	@$(MAKE) package VERSION="$(VERSION)" GOOS=linux GOARCH=amd64
	@$(MAKE) package VERSION="$(VERSION)" GOOS=linux GOARCH=arm64
	@$(MAKE) package VERSION="$(VERSION)" GOOS=darwin GOARCH=amd64
	@$(MAKE) package VERSION="$(VERSION)" GOOS=darwin GOARCH=arm64
	@$(MAKE) package VERSION="$(VERSION)" GOOS=windows GOARCH=amd64
	@$(MAKE) package VERSION="$(VERSION)" GOOS=windows GOARCH=arm64
	@$(MAKE) checksums
	@echo "Release $(VERSION) is ready in $(DIST_DIR)/"

# Generate checksums after all platform archives have been collected in dist/.
checksums:
	@cd $(DIST_DIR) && sha256sum *.tar.gz *.zip > checksums.txt

clean:
	@rm -rf "$(BUILD_DIR)" "$(DIST_DIR)"

help:
	@echo "Targets:"
	@echo "  build       Build the CLI in $(BUILD_DIR)/"
	@echo "  install     Install the CLI with go install"
	@echo "  test        Run all tests"
	@echo "  test-race   Run all tests with the race detector"
	@echo "  fmt         Format Go sources"
	@echo "  lint        Check formatting and run go vet"
	@echo "  check       Run lint and tests"
	@echo "  package     Build a release archive (set VERSION, GOOS, GOARCH)"
	@echo "  release     Build all release archives and checksums (set VERSION)"
	@echo "  checksums   Generate dist/checksums.txt"
	@echo "  clean       Remove local build and release output"
