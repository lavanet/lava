# syntax=docker/dockerfile:1

ARG GO_VERSION="1.20"
ARG RUNNER_IMAGE="gcr.io/distroless/static-debian12:debug"
ARG BUILD_TAGS="netgo,ledger,muslc"

FROM golang:${GO_VERSION}-alpine3.18 AS builder

RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    binutils-gold

WORKDIR /lava

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x
    
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
    -X github.com/cosmos/cosmos-sdk/version.AppName="lavap" \
    -X github.com/cosmos/cosmos-sdk/version.Version=${GIT_VERSION} \
    -X github.com/cosmos/cosmos-sdk/version.Commit=${GIT_COMMIT} \
    -X github.com/cosmos/cosmos-sdk/version.BuildTags=${BUILD_TAGS} \
    -w -s -linkmode=external -extldflags '-Wl,-z,muldefs -static'" \
    -trimpath \
    -o /lava/build/lavap \
    /lava/cmd/lavap/main.go

FROM ${RUNNER_IMAGE}
    
COPY --from=builder /lava/build/lavap /bin/lavap

ENV HOME=/lava
WORKDIR $HOME

# lava api
EXPOSE 1317
# rosetta
EXPOSE 8080
# grpc
EXPOSE 9090
EXPOSE 9091
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

ENTRYPOINT ["lavap"]