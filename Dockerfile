# syntax=docker/dockerfile:1

ARG GO_VERSION="1.20"
ARG BUILD_TAGS="netgo,ledger,muslc"

FROM golang:${GO_VERSION} as builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    build-essential \
    binutils-gold 

WORKDIR /lava

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download

COPY . .

ARG GIT_VERSION
ARG GIT_COMMIT

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    GOWORK=off go build \
    -mod=readonly \
    -tags "netgo,ledger,muslc" \
    -ldflags \
    "-X github.com/cosmos/cosmos-sdk/version.Name="lava" \
    -X github.com/cosmos/cosmos-sdk/version.AppName="lavad" \
    -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION} \
    -X github.com/cosmos/cosmos-sdk/version.Commit=${GIT_COMMIT} \
    -X github.com/cosmos/cosmos-sdk/version.BuildTags=${BUILD_TAGS} \
    -w -s -linkmode=external -extldflags '-Wl,-z,muldefs -static'" \
    -trimpath \
    -o /lava/build/lavavisor \
    /lava/cmd/lavavisor/main.go

FROM ubuntu as builder2

RUN apt-get update && apt-get install -y curl git

ARG TAG

RUN [ -z "$TAG" ] && echo "TAG is required (ex: v0.24.0)" && exit 1 || true

WORKDIR /app

# create bin directory
RUN mkdir -p /app/bin

# download lava binaries
COPY --chmod=777 --from=builder /lava/build/lavavisor /app/bin/lavavisor

RUN curl -L "https://github.com/lavanet/lava/releases/download/${TAG}/lavad-${TAG}-linux-amd64" > /app/bin/lavad && \
    chmod +x /app/bin/lavad  && \
    curl -L "https://github.com/lavanet/lava/releases/download/${TAG}/lavap-${TAG}-linux-amd64" > /app/bin/lavap && \
    chmod +x /app/bin/lavap

###############################################################################
################################## RUNTIME ####################################
###############################################################################
FROM debian:stable

WORKDIR app

RUN apt-get update && apt-get install -y ca-certificates

COPY --from=builder2 /app/bin/* /app/bin/

ENV PATH="/app/bin:${PATH}"
