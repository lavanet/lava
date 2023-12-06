#!/bin/bash

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

############################## JQ INSTALLATION ######################################
# Flag to track if jq installation is successful
jq_installed=false

# Check if jq is not installed
if ! command_exists jq; then
    # Try to install jq using apt (for Debian/Ubuntu)
    if command_exists apt; then
        echo "Installing jq using apt..."
        sudo apt update
        sudo apt install -y jq
        if command_exists jq; then
            echo "jq has been successfully installed."
            jq_installed=true
        fi
    fi

    # Try to install jq using brew (for macOS)
    if ! $jq_installed && command_exists brew; then
        echo "Installing jq using brew..."
        brew install jq
        if command_exists jq; then
            echo "jq has been successfully installed."
            jq_installed=true
        fi
    fi
else
    echo "jq is already installed"
    jq_installed=true
fi

# if jq is still not installed, exit
if ! $jq_installed; then
    echo "Unable to install jq using apt or brew. Please install jq manually."
fi

############################# BUF INSTALLATION ######################################

if ! command_exists buf; then
    # Define the version of buf to install
    BUF_VERSION="1.25.0"  # Update this to the latest version if needed

    # Define the installation directory
    INSTALL_DIR="/usr/local/bin"  # You might need to adjust this based on your preferences

    # Download and install buf
    echo "Downloading buf version $BUF_VERSION..."
    curl -sSL "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-$(uname -s)-$(uname -m)" -o "${INSTALL_DIR}/buf"

    # Add execute permissions to the buf binary
    chmod +x "${INSTALL_DIR}/buf"

    # Check if the installation was successful
    if [ $? -eq 0 ]; then
        echo "buf version $BUF_VERSION has been installed to $INSTALL_DIR."
        exit 0
    else
        echo "An error occurred during the buf installation process. Please install buf manually."
        exit 1
    fi
else
    echo "buf is already installed"
fi

if ! command_exists protoc-gen-gocosmos; then
    git clone https://github.com/cosmos/gogoproto.git
    cd gogoproto
    go mod download
    make install
    cd ..
    rm -rf gogoproto
fi

if ! command_exists yq; then

    if ! check_go_version; then
        echo "Go 1.21 is not installed. Installing..."
        sudo apt install -y golang-1.21
    fi
    go install github.com/mikefarah/yq/v4@latest 

    # Check if the installation was successful
    if [ $? -eq 0 ]; then
        echo "yq version has been installed "
        exit 0
    else
        echo "An error occurred during the yq installation process. Please install buf manually."
        exit 1
    fi
else
    echo "yq is already installed"
fi
