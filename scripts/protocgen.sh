#!/usr/bin/env bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

if ! command_exists buf; then
    echo "buf not found. Please install buf using the init_install.sh script or manually."
    exit 1
fi

set -eo pipefail

echo "Generating gogo proto code"
cd proto
proto_dirs=$(find . -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep "option go_package" $file &> /dev/null ; then
      echo "Generating gogo proto code for $file"
      buf generate --template buf.gen.gogo.yaml $file
    fi
  done
done

cd ..
# move proto files to the right places
cp -r github.com/lavanet/lava/* ./
rm -rf github.com
