#!/bin/bash
source ./scripts/prepare_protobufs.sh

# Flag variable
use_sudo=false

# Parse arguments
while getopts ":s" opt; do
  case $opt in
    s)
      use_sudo=true
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      exit 1
      ;;
  esac
done

# preparing the env
prepare $use_sudo

ROOT_PROTO_DIR="./proto/cosmos/cosmos-sdk"
COSMOS_PROTO_DIR="$ROOT_PROTO_DIR"
LAVA_PROTO_DIR="../../proto"
OUT_DIR="./src/grpc_web_services"

mkdir -p "$OUT_DIR"

protoc \
    --plugin="protoc-gen-js=./node_modules/.bin/protoc-gen-js" \
    --plugin="protoc-gen-ts=./node_modules/.bin/protoc-gen-ts" \
    --js_out="import_style=commonjs,binary:$OUT_DIR" \
    --ts_out="service=grpc-web:$OUT_DIR" \
    --proto_path="$COSMOS_PROTO_DIR" \
    --proto_path="$LAVA_PROTO_DIR" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/relay.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/badges.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/params.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/query.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/provider_payment_storage.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/unique_payment_storage_client_provider.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/subscription/subscription.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/projects/project.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/plans/policy.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/pairing/epoch_payments.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/spec/spec.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/spec/api_collection.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/epochstorage/stake_entry.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/epochstorage/endpoint.proto" \
    "$LAVA_PROTO_DIR/lavanet/lava/conflict/conflict_data.proto" \
    "$COSMOS_PROTO_DIR/gogoproto/gogo.proto" \
    "$COSMOS_PROTO_DIR/google/protobuf/descriptor.proto" \
    "$COSMOS_PROTO_DIR/google/protobuf/wrappers.proto" \
    "$COSMOS_PROTO_DIR/google/api/annotations.proto" \
    "$COSMOS_PROTO_DIR/google/api/http.proto" \
    "$COSMOS_PROTO_DIR/cosmos/base/query/v1beta1/pagination.proto" \
    "$COSMOS_PROTO_DIR/cosmos/base/v1beta1/coin.proto" \
    "$COSMOS_PROTO_DIR/cosmos_proto/cosmos.proto" \
    "$COSMOS_PROTO_DIR//cosmos/staking/v1beta1/staking.proto" \
    "$COSMOS_PROTO_DIR/amino/amino.proto" \

# mv ./src/proto/test ./src/pairing/.
# rm -rf ./src/proto
# mv ./src/pairing ./src/proto


echo "running fix_grpc_web_camel_case.py"
python3 ./scripts/fix_grpc_web_camel_case.py

mkdir -p ./bin/src
cp -r $OUT_DIR ./bin/src/.
