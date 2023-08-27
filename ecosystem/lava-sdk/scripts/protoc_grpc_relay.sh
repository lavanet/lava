
#!/bin/bash

echo "make sure to go mod tidy the lava repo before trying to run this file"

if [[ -z "$GOPATH" ]]; then
    echo "Error: GOPATH is not set. Set the GOPATH environment variable to your Go workspace directory." >&2
    exit 1
fi

if [[ ! -d "$GOPATH" ]]; then
    echo "Error: The directory specified in GOPATH ('$GOPATH') does not exist." >&2
    exit 1
fi

specific_dir="$GOPATH/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.3"

if [[ ! -d "$specific_dir" ]]; then
    echo "Error: The specific directory ('$specific_dir') does not exist under '$GOPATH/pkg/mod'." >&2
    echo "make sure you ran 'go mod tidy' in the lava main repo"
    exit 1
fi


# cosmos is placed in:
# /home/user/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.4/proto/cosmos
# /home/user/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.4/tendermint + amino is there 
# gogo: 
# /home/user/go/pkg/mod/github.com/cosmos/gogoproto@v1.4.10/gogoproto

ROOT_PROTO_DIR="./proto/cosmos/cosmos-sdk"
COSMOS_PROTO_DIR="$ROOT_PROTO_DIR/proto"
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
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/params.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/provider_payment_storage.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/unique_payment_storage_client_provider.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/subscription/subscription.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/projects/project.proto" \
     "$THIRD_PARTY_PROTO_DIR/lavanet/lava/plans/policy.proto" \
     "$THIRD_PARTY_PROTO_DIR/lavanet/lava/pairing/epoch_payments.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/epochstorage/stake_entry.proto" \
    "$THIRD_PARTY_PROTO_DIR/lavanet/lava/epochstorage/endpoint.proto" \
    "$COSMOS_PROTO_DIR/gogoproto/gogo.proto" \
    "$COSMOS_PROTO_DIR/google/protobuf/descriptor.proto" \
    "$COSMOS_PROTO_DIR/google/protobuf/wrappers.proto" \
    "$COSMOS_PROTO_DIR/google/api/annotations.proto" \
    "$COSMOS_PROTO_DIR/google/api/http.proto" \
    "$COSMOS_PROTO_DIR/cosmos/base/v1beta1/coin.proto" \
    "$COSMOS_PROTO_DIR/cosmos/base/query/v1beta1/pagination.proto" \
    "$COSMOS_PROTO_DIR/cosmos_proto/cosmos.proto" \
# mv ./src/proto/test ./src/pairing/.
# rm -rf ./src/proto
# mv ./src/pairing ./src/proto


echo "running fix_grpc_web_camel_case.py"
python3 ./scripts/fix_grpc_web_camel_case.py

cp -r $OUT_DIR ./bin/src/.