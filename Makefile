#!/usr/bin/make -f

# Usage: and targets:
#   [LAVA_BUILD_OPTIONS=...] [ENV...] make [TARGET...]
#
# Targets:
#
#   build		- local build (output: `build/lavad`)
#   docker-build	- docker build (output: `build/lavad`) with docker image
#
#   build-images	- build both amd64,arm64 lavad(s) and docker image(s)
#   build-image-amd64	- docker build (output: `build/lavad-linux-amd64`) with docker image
#   build-image-arm64	- docker build (output: `build/lavad-linux-arm64`) with docker image
#
#   test		- run unit-tests
#   lint		- run the linter
#
# Options:
#   (comma separated list of options to turn on specific features)
#
#   static		- build static binary
#   release		- generate release build
#   nostrip		- do not strip binary from paths
#
#   debug_mutex	- (debug) enable debug mutex
#   mask_consumer_logs	- (debug) enable debug mutex
#
#   cleveldb, rocksdb   - (not to be used)
#
# Environment
#   LAVA_VERSION=...	- select lava version (for 'release')
#   BUILDDIR=...	- select local directory for build output
#   LEDGER_ENABLED	- (not to be used)
#
# Examples:
#
#   Build locally and run unit tests
#     make test build
#
#   Build locally a static binary (runs on any distributions and in containers)
#     LAVA_BUILD_OPTIONS="static" make build
#
#   Build and generate docker image
#     LAVA_BUILD_OPTIONS="static" make docker-build
#
#   Build release [and optionally generate docker image]
#     LAVA_BUILD_OPTIONS="static,release" make build
#     LAVA_BUILD_OPTIONS="static,release" make docker-build
#
#   Build release of specific version, and generate docker image
#     LAVA_VERSION=0.4.3 LAVA_BUILD_OPTIONS="static,release" make docker-build

# do we have .git/ directory?
have_dot_git := $(if $(shell test -d .git && echo true),true,false)

# If we have .git/ directory, then grab the VERSION and COMMIT from git.
# (And if current commit is not tagged, also remove the commit count).
# If we don't have .git/ (like when called from our Makefile), then expect
# VERSION and COMMIT explicitly provided (via BUILD_VERSION, BUILD_COMMIT).

ifeq (true,$(have_dot_git))
  VERSION := $(shell git describe --tags --abbrev=7 --dirty | sed 's/-[0-9]*-g/-/')
  COMMIT := $(shell git log -1 --format='%H')
else
  BUILD_VERSION ?= unknown
  BUILD_COMMIT ?= unknown
  VERSION := $(BUILD_VERSION)
  COMMIT := $(BUILD_COMMIT)
endif

GIT_CLONE := false

# If we have .git/ directory and 'release' option selected, then we consult
# LAVA_VERSION for the desired version; If not set, then we infer the version
# from the currnently checked-out code - and then we examine that it is -
#   - clean (no local uncommitted changes, no untracked files)
#   - matching in version to the desired tag (no extra commit)
#
# Also, if 'release' option selected, then set GIT_COMMIT=true so that that
# the Dockerfile would clone the repository from scratch.

ifeq (true,$(have_dot_git))
  ifeq (release,$(findstring release,$(LAVA_BUILD_OPTIONS)))
    ifneq (,$(LAVA_VERSION))
      VERSION := v$(LAVA_VERSION)
      COMMIT := $(shell git log -1 --format='%H' $(VERSION))
    else
      version_real := $(shell git describe --tags --exact-match 2> /dev/null || echo "none")
      ifneq '$(VERSION)' '$(version_real)'
        $(error Current checked-out code does not match requested release version)
      endif
      ifeq (-dirty,$(findstring -dirty,$(VERSION)))
        $(error Current checked-out code has uncommitted changes or untracked files)
      endif
    endif
    GIT_CLONE := true
  endif
endif

# strip the leading 'v'
VERSION := $(subst v,,$(VERSION))

ifeq (release,$(findstring release,$(LAVA_BUILD_OPTIONS)))
  $(info ----------------------------------------------------------------)
  $(info Building for release: VERSION=$(VERSION) COMMIT=$(COMMIT))
  $(info ----------------------------------------------------------------)
endif

LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
GO_VERSION := $(shell cat go.mod | grep -E 'go [0-9].[0-9]+' | cut -d ' ' -f 2)
DOCKER := $(shell which docker)
BUILDDIR ?= $(CURDIR)/build

export GO111MODULE = on

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq (cleveldb,$(findstring cleveldb,$(LAVA_BUILD_OPTIONS)))
  build_tags += gcc
else ifeq (rocksdb,$(findstring rocksdb,$(LAVA_BUILD_OPTIONS)))
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

null :=
whitespace += $(null) $(null)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# process linker flags

ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=lava \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=lavad \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)"

# For static binaries just set CGO_ENABLED=0.
# (Using only '-link-mode=external -extldflags ...' does not work).
ifeq (static,$(findstring static,$(LAVA_BUILD_OPTIONS)))
  export CGO_ENABLED = 0
endif

ifeq (mask_consumer_logs,$(findstring mask_consumer_logs,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/lavanet/lava/protocol/common.ReturnMaskedErrors=true
endif
ifeq (debug_mutex,$(findstring debug_mutex,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/lavanet/lava/utils.TimeoutMutex=true
endif

ifeq (cleveldb,$(findstring cleveldb,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
else ifeq (rocksdb,$(findstring rocksdb,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif
ifeq (,$(findstring nostrip,$(LAVA_BUILD_OPTIONS)))
  ldflags += -w -s
endif

ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(LAVA_BUILD_OPTIONS)))
  BUILD_FLAGS += -trimpath
endif

###############################################################################
###                                  Build                                  ###
###############################################################################

all: lint test

BUILD_TARGETS := build install

build: BUILD_ARGS=-o $(BUILDDIR)/

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

# build lavad within docker (reproducible) and docker image
docker-build: build-docker-helper build-docker-copier

# build lavad within docker (repducible) for both archs
build-images: build-image_amd64 build-image-arm64

# build lavad-linux-amd64 within docker (reproducible) and docker image
build-image-amd64: TARGETARCH=amd64
build-image-amd64: build-docker-helper build-docker-copier

# build lavad-linux-arm64 within docker (reproducible) and docker image
build-image-arm64: TARGETARCH=arm64
build-image-arm64: build-docker-helper build-docker-copier

RUNNER_IMAGE_DEBIAN := debian:11-slim

define autogen_targetarch
$(if $(TARGETARCH),$(TARGETARCH),$(shell GOARCH= go env GOARCH))
endef

# Note: this target expects TARGETARCH to be defined
build-docker-helper: $(BUILDDIR)/
	$(DOCKER) buildx create --name lavabuilder || true
	$(DOCKER) buildx use lavabuilder
	$(DOCKER) buildx build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		--build-arg GIT_CLONE=$(GIT_CLONE) \
		--build-arg BUILD_OPTIONS=$(LAVA_BUILD_OPTIONS) \
		--build-arg RUNNER_IMAGE=$(RUNNER_IMAGE_DEBIAN) \
		--platform linux/$(call autogen_targetarch) \
		-t lava:$(VERSION) \
		--load \
		-f Dockerfile .
	$(DOCKER) image tag lava:$(VERSION) lava:latest

define autogen_extraver
$(if $(TARGETARCH),-linux-$(TARGETARCH),)
endef

# Note: this target expects TARGETARCH to be defined
build-docker-copier: $(BUILDDIR)/
	$(DOCKER) rm -f lavabinary 2> /dev/null || true
	$(DOCKER) create -ti --name lavabinary lava:$(VERSION)
	$(DOCKER) cp lavabinary:/bin/lavad $(BUILDDIR)/lavad$(call autogen_extraver)
	$(DOCKER) rm -f lavabinary

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

draw-deps:
	@# requires brew install graphviz or apt-get install graphviz
	go get github.com/RobotsAndPencils/goviz
	@goviz -i ./cmd/lavad -d 2 | dot -Tpng -o dependency-graph.png

test:
	@echo "--> Running tests"
	@go test -v ./x/...

lint:
	@echo "--> Running linter"
	golangci-lint run --config .golangci.yml


.PHONY: all build docker-build install lint test \
	go-mod-cache go.sum draw-deps \
	build-docker-helper build-docker-copier \
        build-images build-image-amd64 build-image-arm64 \

