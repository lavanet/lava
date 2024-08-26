#!/bin/sh
set -e
set -o pipefail

echo "### Initializing new Lava node ###"

# Initialize validator if genesis.json doesn't exist
[ ! -f /lava/.lava/config/genesis.json ] && lavad init validator --chain-id "$CHAIN_ID"

# Configure default CLI values
lavad config chain-id "$CHAIN_ID"
lavad config keyring-backend "$KEYRING_BACKEND"
lavad config broadcast-mode sync

# Create users if necessary
[ ! -f /lava/.lava/keyring-test/user1.info ] && lavad keys add user1 || echo "user1 already exists"
[ ! -f /lava/.lava/keyring-test/servicer1.info ] && lavad keys add servicer1 || echo "servicer1 already exists"

# Add genesis accounts
lavad add-genesis-account user1 50000000000000ulava --keyring-backend test || echo "Failed adding user1 as genesis account"
lavad add-genesis-account servicer1 50000000000000ulava --keyring-backend test || echo "Failed adding servicer1 as genesis account"

# Generate signed gentx for servicer1
lavad gentx servicer1 10000000000000ulava --chain-id "$CHAIN_ID" --keyring-backend test || echo "Failed writing signed gen tx for servicer1"

# Register validator
lavad collect-gentxs

echo "### Successfully initialized new Lava node ###"