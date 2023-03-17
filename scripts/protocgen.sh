#!/usr/bin/env bash

set -eo pipefail

# Define a shell function for generating Gogo proto code.
generate_gogo_proto() {
  local dir="$1"
  for file in "$dir"/*.proto; do
    if grep -q go_package "$file"; then
      if command -v buf >/dev/null 2>&1; then
        buf generate --template buf.gen.gogo.yaml "$file"
      else
        echo "Error: buf command not found. See https://docs.buf.build/installation" >&2
        exit 1
      fi
    fi
  done
}

# Install the required protoc execution tools.
go get github.com/regen-network/cosmos-proto/protoc-gen-gocosmos 2>/dev/null

# Clone the Cosmos SDK from GitHub.
# go get github.com/cosmos/cosmos-sdk@v0.45.11 2>/dev/null

# Generate Gogo proto code.
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  generate_gogo_proto "$dir"
done
cd ..

# Move proto files to the right places.
cp -r github.com/lavanet/lava/* ./
rm -rf github.com