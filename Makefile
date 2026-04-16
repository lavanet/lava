#!/usr/bin/make -f

BINDIR ?= $(GOPATH)/bin

# Install both binaries to $GOPATH/bin
install-all: install

install:
	go install -mod=readonly ./cmd/smartrouter
	go install -mod=readonly ./cmd/lavap

install-smartrouter:
	go install -mod=readonly ./cmd/smartrouter

# Build binaries into build/
build-all: build build-lavap

build:
	go build -mod=readonly -o build/smartrouter ./cmd/smartrouter

build-lavap:
	go build -mod=readonly -o build/lavap ./cmd/lavap

# Tests
test:
	go test ./... -count=1 -timeout 300s

test-short:
	go test ./protocol/rpcsmartrouter/... -count=1 -timeout 120s

# Maintenance
tidy:
	go mod tidy

lint:
	go vet ./...

clean:
	rm -rf build/

.PHONY: install install-all install-smartrouter build build-all build-lavap test test-short tidy lint clean
