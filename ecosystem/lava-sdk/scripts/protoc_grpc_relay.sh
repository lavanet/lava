#!/bin/bash
source ./scripts/prepare_protobufs.sh
# preparing the env
prepare

ROOT_PROTO_DIR="./proto/cosmos/cosmos-sdk"
COSMOS_PROTO_DIR="$ROOT_PROTO_DIR"
THIRD_PARTY_PROTO_DIR="../../proto"
OUT_DIR="./src/grpc_web_services"

mkdir -p "$OUT_DIR"

protoc --plugin="protoc-gen-ts=./node_modules/.bin/protoc-gen-ts" \
    --js_out="import_style=commonjs,binary:$OUT_DIR" \
    --ts_out="service=grpc-web:$OUT_DIR" \
    --proto_path="$COSMOS_PROTO_DIR" \
    --proto_path="$THIRD_PARTY_PROTO_DIR" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/relay.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/badges.proto" \

# mv ./src/proto/test ./src/pairing/.
# rm -rf ./src/proto
# mv ./src/pairing ./src/proto


echo "running fix_grpc_web_camel_case.py"
python3 ./scripts/fix_grpc_web_camel_case.py

cp -r $OUT_DIR ./bin/src/.
