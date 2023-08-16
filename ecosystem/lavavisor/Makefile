# Makefile for building Lavavisor
.PHONY: build clean

BINARY_NAME=lavavisor
LDFLAGS=-ldflags "-X main.Version=$(shell git describe --tags --always --dirty)"
GOBIN=$(shell go env GOPATH)/bin

# Detect the OS
ifeq ($(OS),Windows_NT)
    TARGET_OS=windows
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        TARGET_OS=linux
    endif
    ifeq ($(UNAME_S),Darwin)
        TARGET_OS=macos
    endif
endif

build:
ifeq ($(TARGET_OS),windows)
	@echo Building for Windows...
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${GOBIN}/${BINARY_NAME}.exe
endif
ifeq ($(TARGET_OS),linux)
	@echo Building for Linux...
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${GOBIN}/${BINARY_NAME}
endif
ifeq ($(TARGET_OS),macos)
	@echo Building for macOS...
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${GOBIN}/${BINARY_NAME}
endif

clean:
	@echo Cleaning up...
	rm -f ${GOBIN}/${BINARY_NAME}*