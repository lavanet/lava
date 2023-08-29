#!/bin/bash

source ./scripts/prepare_protobufs.sh
# preparing the env
prepare

echo "cloning lavanet proto directory to ./proto/lavanet"
rm -rf ./proto/lavanet
cp -r ../../proto/lavanet ./proto/.

echo "Running ./scripts/Codegen.js"
node ./scripts/codegen.js
echo "building"
npm run build
echo "Script completed."