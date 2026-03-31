#!/usr/bin/make -f

BINDIR ?= $(GOPATH)/bin

install-all: install

install:
	go install -mod=readonly ./cmd/smartrouter
	go install -mod=readonly ./cmd/lavap

install-smartrouter:
	go install -mod=readonly ./cmd/smartrouter

build:
	go build -mod=readonly -o build/smart-router ./cmd/smartrouter

build-lavap:
	go build -mod=readonly -o build/lavap ./cmd/lavap

test:
	go test ./protocol/rpcsmartrouter/... -count=1 -timeout 120s

test-all:
	go test ./... -count=1 -timeout 300s

lint:
	go vet ./...

clean:
	rm -rf build/

.PHONY: install install-all install-smartrouter build build-lavap test test-all lint clean
