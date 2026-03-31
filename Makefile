#!/usr/bin/make -f

BINDIR ?= $(GOPATH)/bin

install-all: install

install:
	go install -mod=readonly ./cmd/lavap

build:
	go build -mod=readonly -o build/lavap ./cmd/lavap

test:
	go test ./protocol/rpcsmartrouter/... -count=1 -timeout 120s

test-all:
	go test ./... -count=1 -timeout 300s

lint:
	go vet ./...

clean:
	rm -rf build/

.PHONY: install install-all build test test-all lint clean
