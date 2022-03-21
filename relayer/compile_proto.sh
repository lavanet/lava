#!/bin/bash

protoc --go_out=. --go-grpc_out=. relayer/proto/relay.proto
