#!/usr/bin/env bash
__dir=$(dirname "$0")
. $__dir/useful_commands.sh

if ! command_exists buf; then
  echo "buf not found."
  echo "Please install buf using the init_install.sh script or manually."
  exit 1
fi

if ! command_exists protoc-gen-gocosmos; then
  echo "protoc-gen-gocosmos not found."
  echo "Please install protoc-gen-gocosmos using the init_install.sh script or manually."
  exit 1
fi

set -e

echo "Generating gogo proto code"
cd proto

proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep -q "option go_package" $file 2> /dev/null ; then
      echo "Generating gogo proto code for $file"
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

cd ..

# move proto files to the right places
cp -r github.com/lavanet/lava/v2/* ./
rm -rf github.com
