#!/bin/bash
set -o errexit -o nounset -o pipefail
command -v shellcheck >/dev/null && shellcheck "$0"

source ./scripts/prepare_protobufs.sh
# preparing the env
prepare

ROOT_PROTO_DIR="./proto/cosmos/cosmos-sdk"
COSMOS_PROTO_DIR="$ROOT_PROTO_DIR"
THIRD_PARTY_PROTO_DIR="../../proto"
OUT_DIR="./src/codec/"

mkdir -p "$OUT_DIR"

protoc \
  --plugin="./node_modules/.bin/protoc-gen-ts_proto" \
  --ts_proto_out="$OUT_DIR" \
  --proto_path="$COSMOS_PROTO_DIR" \
  --proto_path="$THIRD_PARTY_PROTO_DIR" \
  --ts_proto_opt="esModuleInterop=true,forceLong=long,useOptionals=true" \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/epochstorage/stake_entry.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/query.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/relay.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/spec/spec.proto \

# Remove unnecessary codec files
rm -rf \
  src/codec/cosmos_proto/ \
  src/codec/gogoproto/ \
  src/codec/google/api/ \
  src/codec/google/protobuf/descriptor.ts