# syntax = docker/dockerfile:1.2
# WARNING! Use with `docker buildx ...` or `DOCKER_BUILDKIT=1 docker build ...`
# to enable --mount feature used below.

########################################################################
# Dockerfile for reproducible build of lavad binary and docker image
########################################################################

ARG GO_VERSION="1.23"
ARG RUNNER_IMAGE="debian:11-slim"

# --------------------------------------------------------
# Base
# --------------------------------------------------------

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} as base

ARG GIT_VERSION
ARG GIT_COMMIT

# Download debian packages for building
ARG DEBIAN_FRONTEND=noninteractive
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

# set LAVA_BUILD_OPTIONS to control the Makefile behavior.
# (this controls the build options - see there)
ARG BUILD_OPTIONS
ENV LAVA_BUILD_OPTIONS=${BUILD_OPTIONS}

# set LAVA_BINARY to control Makefile behavior.
# (this controls which binaries will be generated - see there)
ARG LAVA_BINARY
ENV LAVA_BINARY=${LAVA_BINARY}

# Download go dependencies
WORKDIR /lava
COPY go.mod go.sum ./
RUN --mount=type=cache,sharing=private,target=/root/.cache/go-build \
    --mount=type=cache,sharing=private,target=/go/pkg/mod \
    go mod download

# Copy the remaining files
COPY . .

# Git clone the sources if requested
# NOTE TODO: after reset of chain (lava-testnet-1) prefix 'v' to ${GIT_VERSION}
RUN if [ "${GIT_CLONE}" = true ]; then \
    find . -mindepth 1 -delete && \
    git clone --depth 1 --branch v${GIT_VERSION} https://github.com/lavanet/lava . \
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

# Download IP geolocation databases.
# curl flags: -f fails (non-zero exit) on HTTP errors instead of silently
# saving the error page as the "database"; -S surfaces the error; -L follows
# redirects; -s hides the progress meter.
# Optionally pin SHA-256 checksums via build args to detect tampering or
# corruption in transit. They default to empty (verification skipped) because
# the ip2asn dataset is refreshed upstream frequently; operators can pin a
# known-good digest at build time, e.g. --build-arg COUNTRIES_SHA256=<hash>.
ARG IP2ASN_SHA256=""
ARG COUNTRIES_SHA256=""

RUN curl -fsSL https://iptoasn.com/data/ip2asn-v4.tsv.gz -o /tmp/ip2asn-v4.tsv.gz \
    && if [ -n "$IP2ASN_SHA256" ]; then echo "$IP2ASN_SHA256  /tmp/ip2asn-v4.tsv.gz" | sha256sum -c -; fi \
    && gunzip /tmp/ip2asn-v4.tsv.gz \
    && test -s /tmp/ip2asn-v4.tsv

RUN curl -fsSL https://storage.googleapis.com/lavanet-public-asssets/countries.csv -o /tmp/countries.csv \
    && if [ -n "$COUNTRIES_SHA256" ]; then echo "$COUNTRIES_SHA256  /tmp/countries.csv" | sha256sum -c -; fi \
    && test -s /tmp/countries.csv

# Build lavad binary
RUN --mount=type=cache,sharing=private,target=/root/.cache/go-build \
    --mount=type=cache,sharing=private,target=/go/pkg/mod \
    LAVA_BUILD_OPTIONS="${LAVA_BUILD_OPTIONS},static" make build

# --------------------------------------------------------
# Cosmovisor
# --------------------------------------------------------

FROM --platform=$BUILDPLATFORM base as cosmovisor

WORKDIR /lava

# Download Cosmovisor
RUN --mount=type=cache,sharing=private,target=/root/.cache/go-build \
    --mount=type=cache,sharing=private,target=/go/pkg/mod \
    go mod init disposable \
    && go get github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor@v1.0.0 \
    && go install github.com/cosmos/cosmos-sdk/cosmovisor/cmd/cosmovisor@v1.0.0


# --------------------------------------------------------
# Runner-base
# --------------------------------------------------------

# Download debian packages for runner

FROM ${RUNNER_IMAGE} as runner-base

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install -yq --no-install-recommends \
    git curl unzip ca-certificates jq \
    && apt-get -y purge \
    && apt-get -y clean \
    && apt-get -y autoremove \
    && rm -rf /var/lib/apt/lists/*

# --------------------------------------------------------
# Runner
# --------------------------------------------------------

FROM runner-base

ARG LAVA_BINARY
COPY --from=cosmovisor --chown=0:0 --chmod=755 /go/bin/cosmovisor /bin/
COPY --from=builder --chown=0:0 --chmod=755 /lava/build/* /bin/

ENV HOME /lava
WORKDIR $HOME

COPY docker/entrypoint.sh /
COPY docker/start_node.sh start_node.sh
COPY docker/start_portal.sh start_portal.sh

COPY --from=builder --chown=0:0 --chmod=755 /tmp/ip2asn-v4.tsv ./config/badge/ip2asn-v4.tsv
COPY --from=builder --chown=0:0 --chmod=755 /tmp/countries.csv ./config/badge/countries.csv

# common setup
ENV LAVA_COSMOVISOR_URL=
ENV LAVA_CONFIG_GIT_URL=
ENV LAVA_CHAIN_ID=
ENV LAVA_MONIKER=

# common runtime
ENV LAVA_LOG_LEVEL=

# provider/validator [OUTDATED]
#ENV LAVA_ACCOUNT=
#ENV LAVA_USER=
#ENV LAVA_ADDRESS=
#ENV LAVA_KEYRING=
#ENV LAVA_STAKE_AMOUNT=
#ENV LAVA_GAS_MODE=
#ENV LAVA_GAS_ADJUST=
#ENV LAVA_GAS_PRICE=
#ENV LAVA_GEOLOCATION=
#ENV LAVA_RPC_NODE=
#ENV LAVA_LISTEN_IP=
#ENV LAVA_NODE_PORT_API=
#ENV LAVA_NODE_PORT_GRPC=
#ENV LAVA_NODE_PORT_GRPC_WEB=
#ENV LAVA_NODE_PORT_P2P=
#ENV LAVA_NODE_PORT_RPC=
#ENV LAVA_PORTAL_PORT=
#ENV LAVA_RELAY_CHAIN_ID=
#ENV LAVA_RELAY_IFACE=
#ENV LAVA_RELAY_NODE_URL=
#ENV LAVA_RELAY_ENDPOINT=

# lava api
EXPOSE 1317
# rosetta
EXPOSE 8080
# grpc
EXPOSE 9090
# grpc-web
EXPOSE 9090
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

ENTRYPOINT ["/entrypoint.sh"]