#!/bin/bash
set -o errexit -o nounset -o pipefail
command -v shellcheck >/dev/null && shellcheck "$0"

ROOT_PROTO_DIR="./proto/cosmos/cosmos-sdk"
COSMOS_PROTO_DIR="$ROOT_PROTO_DIR/proto"
THIRD_PARTY_PROTO_DIR="../../proto"
OUT_DIR="./src/codec/"

mkdir -p "$OUT_DIR"

protoc \
  --plugin="./node_modules/.bin/protoc-gen-ts_proto" \
  --ts_proto_out="$OUT_DIR" \
  --proto_path="$COSMOS_PROTO_DIR" \
  --proto_path="$THIRD_PARTY_PROTO_DIR" \
  --ts_proto_opt="esModuleInterop=true,forceLong=long,useOptionals=true" \
  "$COSMOS_PROTO_DIR/cosmos/auth/v1beta1/auth.proto" \
  "$COSMOS_PROTO_DIR/cosmos/auth/v1beta1/query.proto" \
  "$COSMOS_PROTO_DIR/cosmos/bank/v1beta1/bank.proto" \
  "$COSMOS_PROTO_DIR/cosmos/bank/v1beta1/query.proto" \
  "$COSMOS_PROTO_DIR/cosmos/bank/v1beta1/tx.proto" \
  "$COSMOS_PROTO_DIR/cosmos/base/abci/v1beta1/abci.proto" \
  "$COSMOS_PROTO_DIR/cosmos/base/query/v1beta1/pagination.proto" \
  "$COSMOS_PROTO_DIR/cosmos/base/v1beta1/coin.proto" \
  "$COSMOS_PROTO_DIR/cosmos/crypto/multisig/v1beta1/multisig.proto" \
  "$COSMOS_PROTO_DIR/cosmos/crypto/secp256k1/keys.proto" \
  "$COSMOS_PROTO_DIR/cosmos/distribution/v1beta1/distribution.proto" \
  "$COSMOS_PROTO_DIR/cosmos/distribution/v1beta1/query.proto" \
  "$COSMOS_PROTO_DIR/cosmos/distribution/v1beta1/tx.proto" \
  "$COSMOS_PROTO_DIR/cosmos/staking/v1beta1/query.proto" \
  "$COSMOS_PROTO_DIR/cosmos/staking/v1beta1/staking.proto" \
  "$COSMOS_PROTO_DIR/cosmos/staking/v1beta1/tx.proto" \
  "$COSMOS_PROTO_DIR/cosmos/tx/signing/v1beta1/signing.proto" \
  "$COSMOS_PROTO_DIR/cosmos/tx/v1beta1/tx.proto" \
  "$COSMOS_PROTO_DIR/cosmos/vesting/v1beta1/vesting.proto" \
  "$COSMOS_PROTO_DIR/tendermint/abci/types.proto" \
  "$COSMOS_PROTO_DIR/tendermint/crypto/keys.proto" \
  "$COSMOS_PROTO_DIR/tendermint/crypto/proof.proto" \
  "$COSMOS_PROTO_DIR/tendermint/libs/bits/types.proto" \
  "$COSMOS_PROTO_DIR/tendermint/types/params.proto" \
  "$COSMOS_PROTO_DIR/tendermint/types/types.proto" \
  "$COSMOS_PROTO_DIR/tendermint/types/validator.proto" \
  "$COSMOS_PROTO_DIR/tendermint/version/types.proto" \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/conflict/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/epochstorage/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/spec/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/common/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/plans/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/projects/*.proto \
  $THIRD_PARTY_PROTO_DIR/lavanet/lava/subscription/*.proto \

# Remove unnecessary codec files
rm -rf \
  src/codec/cosmos_proto/ \
  src/codec/gogoproto/ \
  src/codec/google/api/ \
  src/codec/google/protobuf/descriptor.ts