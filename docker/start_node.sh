#!/bin/bash
# vim:sw=4:ts=4:et

set -e

SETUP_CONFIG_GIT_URL='https://github.com/K433QLtr6RA9ExEq/GHFkqmTzpdNLDd6T.git'
COSMOVISOR_ZIP_URL='https://lava-binary-upgrades.s3.amazonaws.com/testnet/cosmovisor-upgrades/cosmovisor-upgrades.zip'

debug() {
    echo "DBG: $@"
}

error() {
    echo "ERR: $@"
    exit 1
}

setup_node() {
    setup_config_dir=$(basename "$SETUP_CONFIG_GIT_URL")

    # Remove old data (if any)
    rm -rf "$setup_config_dir"

    # Download setup configuration
    git clone --depth 1 "$SETUP_CONFIG_GIT_URL" "$setup_config_dir" || \
        error "setup: failed to clone setup configuration"

    cd "$setup_config_dir/testnet-1"
    source setup_config/setup_config.sh

    # Keep a copy handy for when we restart
    cp setup_config/setup_config.sh "$HOME"

    # Remove old config (if any)
    rm -rf "$lavad_home_folder"

    # Copy initial configuration and genesis data
    mkdir -p "$lavad_home_folder" "$lava_config_folder"
    cp default_lavad_config_files/* "$lava_config_folder"
    cp genesis_json/genesis.json "$lava_config_folder"
}

setup_env() {
    # Environment variables for cosmovisor
    export DAEMON_NAME=lavad
    export CHAIN_ID=lava-testnet-1
    export DAEMON_HOME="$HOME/.lava"
    export DAEMON_ALLOW_DOWNLOAD_BINARIES=true
    export DAEMON_LOG_BUFFER_SIZE=512
    export DAEMON_RESTART_AFTER_UPGRADE=true
    export UNSAFE_SKIP_BACKUP=true
}

# Note: expected to run in the setup config directory - see setup_node()
setup_cosmovisor() {
    # Download latest cosmovisor-upgrades
    curl -L --progress-bar -o cosmovisor-upgrades.zip "$COSMOVISOR_ZIP_URL" || \
        error "setup: failed to download cosmovisor upgrades"
    unzip -o cosmovisor-upgrades.zip || \
        error "setup: failed to unzip cosmovisor upgrades"

    # Copy cosmovisor configuration
    mkdir -p "$lavad_home_folder"/cosmovisor
    cp -r cosmovisor-upgrades/* "$lavad_home_folder"/cosmovisor

    # Initialize the chain
    output=$(
        "${lavad_home_folder}/cosmovisor/genesis/bin/lavad" init \
        my-node \
        --chain-id lava-testnet-1 \
        --home "$lavad_home_folder" \
        --overwrite
    )

    # An error message about missing upgrade-info.json is expected;
    # anything else is unexpected and should abort.
    if [ $? -ne 0 ]; then
        case "$output" in
        "*upgrade-info.json: no such file or directory*") ;;
        "*") error "setup: failed to initialize the chain" ;;
        esac
    fi

    # Copy genesis data again
    cp genesis_json/genesis.json "$lava_config_folder"/genesis.json
}

setup_env

if [ ! -e "$HOME/.lava/cosmovisor/current" ]; then
    setup_node
    setup_cosmovisor
else
    source "$HOME/setup_config.sh"
fi

exec /bin/cosmovisor start --home="$lavad_home_folder" --p2p.seeds "$seed_node"
