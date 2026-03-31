# syntax=docker/dockerfile:1

ARG GO_VERSION="1.23"
ARG ALPINE_VERSION="3.21"
ARG RUNNER_IMAGE="gcr.io/distroless/static-debian12:debug"

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

RUN apk add --no-cache \
    ca-certificates \
    build-base \
    linux-headers \
    binutils-gold

WORKDIR /smart-router

ENV GOCACHE=/root/.cache/go-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    --mount=type=bind,source=./go.sum,target=/smart-router/go.sum \
    --mount=type=bind,source=./go.mod,target=/smart-router/go.mod \
    go mod download -x

COPY . .

ARG GIT_VERSION="dev"
ARG GIT_COMMIT="unknown"

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg/mod \
    GOWORK=off go build \
    -mod=readonly \
    -tags "netgo,muslc" \
    -ldflags \
    "-X main.version=${GIT_VERSION} \
    -X main.commit=${GIT_COMMIT} \
    -w -s -linkmode=external -extldflags '-Wl,-z,muldefs -static'" \
    -trimpath \
    -o /smart-router/build/smart-router \
    /smart-router/cmd/smartrouter/main.go

FROM ${RUNNER_IMAGE}

COPY --from=builder /smart-router/build/smart-router /bin/smart-router

ENV HOME=/smart-router
WORKDIR $HOME

# smart router listener
EXPOSE 3360
# metrics
EXPOSE 7779

ENTRYPOINT ["smart-router"]
