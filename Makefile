.DEFAULT_GOAL := help

GO ?= go
CLI_BINARY ?= ./cli/pocketbase-pockethost
DIST_DIR ?= ./dist
VERSION ?= dev

.PHONY: help check test vet build build-all install clean

help:
	@echo "Available targets:"
	@echo "  make check     Run tests and static analysis"
	@echo "  make test      Run Go unit tests"
	@echo "  make vet       Run go vet"
	@echo "  make build     Build the local CLI binary"
	@echo "  make build-all Build six CGO-free platform binaries"
	@echo "  make install   Install the CLI with go install"
	@echo "  make clean     Remove local build artifacts"

check: test vet

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	CGO_ENABLED=0 $(GO) build -o $(CLI_BINARY) ./cli/cmd/pocketbase-pockethost

build-all:
	@mkdir -p $(DIST_DIR)
	@for target in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64 windows-arm64; do \
		GOOS=$${target%-*}; \
		GOARCH=$${target#*-}; \
		output="$(DIST_DIR)/pocketbase-pockethost_$(VERSION)_$${GOOS}_$${GOARCH}"; \
		if [ "$${GOOS}" = "windows" ]; then output="$${output}.exe"; fi; \
		echo "Building $${GOOS}/$${GOARCH}"; \
		CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} $(GO) build -o "$${output}" ./cli/cmd/pocketbase-pockethost; \
	done

install:
	$(GO) install ./cli/cmd/pocketbase-pockethost

clean:
	rm -rf $(DIST_DIR) $(CLI_BINARY)
