# syntax = docker/dockerfile:1.2
# WARNING! Use with `docker buildx ...` or `DOCKER_BUILDKIT=1 docker build ...`
# to enable --mount feature used below.

########################################################################
# Dockerfile for reproducible build of lavad binary and docker image
########################################################################

ARG GO_VERSION="1.18.2"
ARG RUNNER_IMAGE="debian:11-slim"

# --------------------------------------------------------
# Base
# --------------------------------------------------------

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} as base

ARG GIT_VERSION
ARG GIT_COMMIT

# Download debian packages for building
RUN --mount=type=cache,target=/var/cache/apt \
    rm -f /etc/apt/apt.conf.d/docker-clean && \
    apt-get update && \
    apt-get install -yqq --no-install-recommends \
        build-essential \
        ca-certificates \
        curl

# --------------------------------------------------------
# Builder
# --------------------------------------------------------

FROM --platform=$BUILDPLATFORM base as builder

ARG TARGETOS
ARG TARGETARCH

# set GIT_CLONE=true to force 'git clone' of sources from repository
# (useful to compile a specific version, combined with GIT_VERSION).
ARG GIT_CLONE=false

# Download go dependencies
WORKDIR /lava
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy the remaining files
COPY . .

# Git clone the sources if requested
# NOTE TODO: after reset of chain (lava-testnet-1) prefix 'v' to ${GIT_VERISON}
RUN if [ "${GIT_CLONE}" = true ]; then \
      find . -mindepth 1 -delete && \
      git clone --depth 1 --branch ${GIT_VERSION} https://github.com/lavanet/lava . \
    ; fi

# Remove tag v0.4.0 (same v0.4.0-rc2, which was used in the upgrade proposal
# and must always be reported) to not eclipse the v0.4.0-rc2 tag.
# NOTE TODO: after reset of chain (lava-testnet-1) remove this
RUN git tag -d v0.4.0 || true

# Fix glitch in Makefile for versions < 0.4.3
# NOTE TODO: after reset of chain (lava-testnet-1) remove this
RUN sed -i 's/whitespace += $(whitespace)/whitespace := $(whitespace) $(whitespace)/g' Makefile

# Export our version/commit for the Makefile to know (the .git directory
# is not here, so the Makefile cannot infer them).
ENV BUILD_VERSION=${GIT_VERSION}
ENV BUILD_COMMIT=${GIT_COMMIT}

ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}

# Build lavad binary
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    LAVA_BUILD_OPTIONS="static" make build

# --------------------------------------------------------
# Cosmovisor
# --------------------------------------------------------

FROM --platform=$BUILDPLATFORM builder as cosmovisor

# Download Cosmovisor
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go get github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor@v1.0.0 \
    && go install github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor@v1.0.0


# --------------------------------------------------------
# Runner-base
# --------------------------------------------------------

# Download debian packages for runner

FROM ${RUNNER_IMAGE}

COPY --from=builder /lava/build/lavad /bin/lavad

ENV HOME /lava
WORKDIR $HOME

ENTRYPOINT ["/bin/lavad"]
