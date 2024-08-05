#!/usr/bin/make -f

# Usage: and targets:
#   [LAVA_BUILD_OPTIONS=...] [ENV...] make [TARGET...]
#
# Binaries:
#   lavad               - binary to run for nodes and validators
#   lavap               - binary to run for providers
#
# Targets:
#   (the output of builds goes to $(BUILD_DIR), by default: build/*)
#
#   build-[BIN]         - local build of select binary; [BIN] is one of "Binaries" (or "all")
#   build               - equivalent to build-$(LAVA_BINARY) if set (if not set: error)
#
#   install-[BIN]       - local install of select binary; [BIN] is one of "Binaries" (or "all")
#   install             - equivalent to install-$(LAVA_BINARY) if set (if not set: error)
#
#   docker-build-[BIN]  - docker build of select binary; [BIN] is one of "Binaries" (or "all")
#   docker-build        - equivalent to docket-build-$(LAVA_BINARY) if set (if not set: error)
#
#   build-images        - build both amd64,arm64 lavad(s) and docker image(s)
#   build-image-amd64   - docker build (output: `$(BUILD_DIR)/*-linux-amd64`) with docker image
#   build-image-arm64   - docker build (output: `$(BUILD_DIR)/*-linux-arm64`) with docker image
#
#   proto-gen           - (re)generate protobuf file
#   test                - run unit-tests
#   lint                - run the linter
#
# Options:
#   (comma separated list of options to turn on specific features)
#
#   static	            - build static binary
#   release	            - generate release build
#   nostrip	            - do not strip binary from paths
#
#   debug_mutex         - (debug) enable debug mutex
#   mask_consumer_logs  - (debug) enable debug mutex
#
#   cleveldb, rocksdb   - (not to be used)
# 
#   production-log-level - logs for errros related to bad usage (rather than bugs) set as warnings
#
# Environment
#   LAVA_VERSION=...    - select lava version (for 'release')
#   LAVA_BINARY=...     - select binary to build: "lavad", "lavap", or "all"
#   BUILDDIR=...        - select local directory for build output
#   LEDGER_ENABLED      - (not to be used)
#
# Examples:
#
#   Build locally and run unit tests
#     make test build
#
#   Build locally static binaries (runs on any distributions and in containers)
#     LAVA_BUILD_OPTIONS="static" make build-all
#
#   Build and generate docker image
#     LAVA_BUILD_OPTIONS="static" make docker-build-all
#
#   Build release [and optionally generate docker image]
#     LAVA_BUILD_OPTIONS="static,release" make build-all
#     LAVA_BUILD_OPTIONS="static,release" make docker-build-all
#
#   Build release of specific version, and generate docker image
#     LAVA_VERSION=0.4.3 LAVA_BUILD_OPTIONS="static,release" make docker-build

# in order to print lava build options set: 
# $(info LAVA_BUILD_OPTIONS is set to: $(LAVA_BUILD_OPTIONS))

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

# trick: whitespace hold a single whitespace
null :=
whitespace += $(null) $(null)
comma := ,
build_tags_comma_sep := $(subst $(whitespace),$(comma),$(build_tags))

# trick: newline hold a single newline (Makefile ignores first newline)
define newline


endef

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
  ldflags += -X github.com/lavanet/lava/v2/protocol/common.ReturnMaskedErrors=true
endif
ifeq (debug_mutex,$(findstring debug_mutex,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/lavanet/lava/v2/utils.TimeoutMutex=true
endif

ifeq (cleveldb,$(findstring cleveldb,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
else ifeq (rocksdb,$(findstring rocksdb,$(LAVA_BUILD_OPTIONS)))
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=rocksdb
endif

ifeq (release,$(findstring release,$(LAVA_BUILD_OPTIONS)))
  $(info Building With Production Flag)
  ldflags += -X github.com/lavanet/lava/v2/utils.ExtendedLogLevel=production
endif

ifeq (debug_payment_e2e,$(findstring debug_payment_e2e,$(LAVA_BUILD_OPTIONS)))
  $(info Building With Debug Payment E2E Flag. Making a month 2 minutes)
  ldflags += -X github.com/lavanet/lava/v2/utils.DebugPaymentE2E=debug_payment_e2e
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

LAVA_ALL_BINARIES := lavad lavap lavavisor

# helper target/build functions

# return prefix of $2 before first occurence of $1
# example: $(call prefix -,hello-world) yields "hello"
define prefix
$(word 1,$(subst $1,$(whitespace),$2))
endef

# generate a dependency of each of (matrix) $1-$2: $3 (where $1 is a list)
# example: $(call makedep,x y,suffix,depends) yields "x-suffix y-suffix: depends"
define makedep
$(strip $(foreach t,$1,$(t)-$(2) )): $3
endef

###############################################################################
###                                  Build                                  ###
###############################################################################

all: lint test

# test that $(LAVA_BINARY) is valid: wrap valid values within ":" to avoid
# sub-matches (and explicitly rule out the full combo of all valid binaries
# as a single string).

valid_binaries := :$(subst $(whitespace),:,$(LAVA_ALL_BINARIES)):all:

# arg should not contain ":"
# :arg: should appear in valid_binaries
define validate_binary
$(if $(findstring :,$1),$(error $2),)
$(if $(findstring :$1:,$(valid_binaries)),,$(error $2))
endef

# targets "build","install" (standalone) are only possible if LAVA_BINARY is
# defined - either one of available binaries in $(LAVA_ALL_BINARIES) or "all";
# otherwise it is an error.

ifeq (,$(LAVA_BINARY))
LAVA_BINARY := bad
# these will be called if target is "build" or "install" and $(LAVA_BINARY)
# remains undefined.
build-bad install-bad docker-build-bad:
	$(error "target '$(call prefix,-,$@)' requires valid env 'LAVA_BINARY'")
else
$(call validate_binary,$(LAVA_BINARY),\
  "targets 'build'(comma) 'install' require valid env 'LAVA_BINARY'")
endif

build: build-$(LAVA_BINARY)
install: install-$(LAVA_BINARY)

build-%: BUILD_ARGS=-o $(BUILDDIR)/
build-% install-%: $(BUILDDIR)/
	$(call validate_binary,$*,"No rule to make target '$@'")
	go $(call prefix,-,$@) -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./cmd/$*

# dummy rule to prevent the above wildcard from catching {build,install}-all
$(call makedep,build install,all,)
	@echo -n

build-all: $(foreach binary,$(LAVA_ALL_BINARIES),build-$(binary))
install-all: $(foreach binary,$(LAVA_ALL_BINARIES),install-$(binary))

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

docker-build: docker-build-$(LAVA_BINARY)

# build lavad within docker (reproducible) and docker image
docker-build-%: build-docker-helper-% build-docker-copier-%
	@echo -n

# build lavad within docker (repducible) for both archs
build-images: build-image_amd64 build-image-arm64

# build lavad-linux-amd64 within docker (reproducible) and docker image
build-image-amd64: TARGETARCH=amd64
build-image-amd64: build-docker-helper-all build-docker-copier-all

# build lavad-linux-arm64 within docker (reproducible) and docker image
build-image-arm64: TARGETARCH=arm64
build-image-arm64: build-docker-helper-all build-docker-copier-all

RUNNER_IMAGE_DEBIAN := debian:11-slim

define autogen_targetarch
$(if $(TARGETARCH),$(TARGETARCH),$(shell GOARCH= go env GOARCH))
endef

# Note: this target expects TARGETARCH to be defined
build-docker-helper-%: $(BUILDDIR)/
	$(call validate_binary,$*,"No rule to make target '$@'")
	$(DOCKER) buildx create --name lavabuilder || true
	$(DOCKER) buildx use lavabuilder
	$(DOCKER) buildx build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg GIT_VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(COMMIT) \
		--build-arg GIT_CLONE=$(GIT_CLONE) \
		--build-arg BUILD_OPTIONS=$(LAVA_BUILD_OPTIONS) \
		--build-arg RUNNER_IMAGE=$(RUNNER_IMAGE_DEBIAN) \
		--build-arg LAVA_BINARY=$* \
		--platform linux/$(call autogen_targetarch) \
		-t lava:$(VERSION) \
		--load \
		-f Dockerfile .
	$(DOCKER) image tag lava:$(VERSION) lava:latest

define autogen_extraver
$(if $(TARGETARCH),-linux-$(TARGETARCH),)
endef

define expand_lava_binary
$(subst all,$(LAVA_ALL_BINARIES),$1)
endef

# Note: this target expects TARGETARCH to be defined
build-docker-copier-%: $(BUILDDIR)/
	$(call validate_binary,$*,"No rule to make target '$@'")
	@$(DOCKER) rm -f lavabinary 2> /dev/null || true
	@$(DOCKER) create -ti --name lavabinary lava:$(VERSION) > /dev/null
	$(foreach binary,$(call expand_lava_binary,$*),\
	  $(DOCKER) cp lavabinary:/bin/$(binary) $(BUILDDIR)/$(binary)$(call autogen_extraver) $(newline))
	@$(DOCKER) rm -f lavabinary > /dev/null

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

# protobuf generation

define rwildcard
  $(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))
endef

protoVer=0.13.5
protoImageName=ghcr.io/cosmos/proto-builder:$(protoVer)
containerProtoGen=lava-proto-gen-$(protoVer)
containerProtoFmt=lava-proto-fmt-$(protoVer)

proto-gen: $(BUILDDIR)/proto-gen
$(BUILDDIR)/proto-gen: $(call rwildcard,proto/,*.proto)
	@echo "Generating protobuf files"
	docker run --rm -v $(CURDIR):/workspace --workdir /workspace \
		$(protoImageName) sh ./scripts/protocgen.sh
	@touch $@

.PHONY: all docker-build lint test proto-gen \
	go-mod-cache go.sum draw-deps \
	build build-all install install-all \
	build-images build-image-amd64 build-image-arm64
