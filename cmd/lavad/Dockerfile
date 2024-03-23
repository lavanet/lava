# syntax = docker/dockerfile:1.2

ARG GO_VERSION="1.21"
ARG RUNNER_IMAGE="ubuntu:22.04"

FROM golang:${GO_VERSION}-alpine3.18 as builder

RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers

WORKDIR /lava
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    GOWORK=off go build \
    -mod=readonly \
    -ldflags \
    "-X github.com/cosmos/cosmos-sdk/version.Name="lava" \
    -X github.com/cosmos/cosmos-sdk/version.AppName="lavad" \
    -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION} \
    -X github.com/cosmos/cosmos-sdk/version.Commit=${GIT_COMMIT} \
    -w -s -linkmode=external -extldflags '-Wl,-z,muldefs -static'" \
    -trimpath \
    -o /lava/build/lavad \
    /lava/cmd/lavad/main.go

FROM ${RUNNER_IMAGE}

ARG COSMOVISOR_VERSION="v1.5.0"

RUN set -eux && \
    apt-get update && \
    apt-get install -y curl && \
    curl -sSL https://github.com/cosmos/cosmos-sdk/releases/download/cosmovisor%2F${COSMOVISOR_VERSION}/cosmovisor-${COSMOVISOR_VERSION}-linux-amd64.tar.gz | \
    tar -xz -C /usr/local/bin cosmovisor

RUN set -eux && \
    chmod +x /usr/local/bin/* && \
    groupadd -g 1000 cosmovisor && \
    useradd -u 1000 -g 1000 -s /bin/bash -Md /lava cosmovisor

ENV HOME /lava
WORKDIR $HOME

RUN mkdir -p ~/.lava/cosmovisor && \
    mkdir -p ~/.lava/cosmovisor/genesis && \
    mkdir -p ~/.lava/cosmovisor/genesis/bin && \
    mkdir -p ~/.lava/cosmovisor/upgrades

ENV LAVA_MONIKER="my-docker-lavad"

ENV DAEMON_NAME=lavad \
    DAEMON_HOME=/lava/.lava \
    CHAIN_HOME=/lava \
    UNSAFE_SKIP_BACKUP=true \
    DAEMON_RESTART_AFTER_UPGRADE=true \
    DAEMON_ALLOW_DOWNLOAD_BINARIES=true \
    DAEMON_LOG_BUFFER_SIZE=512 \
    LAVA_CHAIN_ID=lava-testnet-2

COPY --from=builder /lava/build/lavad /lava/.lava/cosmovisor/genesis/bin/lavad

RUN /lava/.lava/cosmovisor/genesis/bin/lavad init ${LAVA_MONIKER} \
      --chain-id ${LAVA_CHAIN_ID} \
      --home /lava/.lava \
      --overwrite

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

CMD [ "cosmovisor", "run", "start", "--log_level", "warn" ]