#!/bin/bash

__lava_root_dir=$(realpath $( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )/../../..)
__scripts_dir=$__lava_root_dir/scripts
source $__scripts_dir/useful_commands.sh

# Check for GOTPATH
if [[ -z "$GOPATH" ]]; then
    echo ">>> GOPATH is not set. Exiting..."
    exit 1
fi

# Check for Python 3
if ! command_exists python3; then
    echo ">>> python3 is not installed. Exiting..."
    exit 1
fi

# Install npm if needed
if ! command_exists npm; then
    # Try to install npm using apt
    echo ">>> Installing npm using apt..."
    sudo apt update
    sudo apt install -y npm
    if ! command_exists npm; then
        echo ">>> Failed to install npm. Exiting..."
        exit 1
    fi
else
    echo ">>> npm is installed"
fi

# Install yarn if needed
if ! command_exists yarn; then
    # Try to install yarn using npm
    echo ">>> Installing yarn using npm..."
    npm install -g yarn
    if ! command_exists yarn; then
        echo ">>> Failed to install yarn. Exiting..."
        exit 1
    fi
else
    echo ">>> yarn is installed"
fi

echo ">>> Running yarn to install packages..."
yarn
if command_exists yarn; then
    echo ">>> All packages have been successfully installed."
fi

# Run go mod tidy in lava root dir 
cd __lava_root_dir
go mod tidy
cd -

# Install the protobuf compiler if needed
if ! command_exists protoc; then
    # Try to install protoc using apt
    echo ">>> Installing protoc using apt..."
    sudo apt update
    sudo apt install -y protobuf-compiler
    if ! command_exists protoc; then
        echo ">>> Failed to install protobuf-compiler. Exiting..."
        exit 1
    fi
else
    echo ">>> protoc is installed"
fi

# Install the ts-protoc-gen plugin if needed
if ! command_exists ts-protoc-gen; then
    # Try to install ts-protoc-gen using npm
    echo ">>> Installing ts-protoc-gen using npm..."
    npm install ts-protoc-gen
    if npm list ts-protoc-gen | grep -q "ts-protoc-gen"; then
        echo ">>> ts-protoc-gen is installed"
    else
        echo ">>> ts-protoc-gen is not installed"
        exit 1
    fi
else
    echo ">>> ts-protoc-gen is installed"
fi

# Run the gRPC generation script
./scripts/protoc_grpc_relay.sh
