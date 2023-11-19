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
prepare

echo "cloning lavanet proto directory to ./proto/lavanet"
rm -rf ./proto/lavanet
cp -r ../../proto/lavanet ./proto/.

echo "Running ./scripts/Codegen.js"
node ./scripts/codegen.js
echo "building"
npm run build
echo "Script completed."