#!/bin/sh
set -e
set -o pipefail

echo "### Initializing new Lava node ###"
echo "Initializing new validator"
[ ! -f /lava/.lava/config/genesis.json ] && lavad init validator --chain-id $CHAIN_ID
echo "Configuring default CLI values"
lavad config chain-id $CHAIN_ID
lavad config keyring-backend $KEYRING_BACKEND
lavad config broadcast-mode sync
echo "Creating users"
[ ! -f /lava/.lava/keyring-test/user1.info ] && lavad keys add user1 || echo user1 already exists
[ ! -f /lava/.lava/keyring-test/servicer1.info ] && lavad keys add servicer1 || echo servicer1 already exists
lavad add-genesis-account user1 50000000000000ulava --keyring-backend test || echo failed adding user1 as genesis account
lavad add-genesis-account servicer1 50000000000000ulava --keyring-backend test || echo failed servicer1 as genesis account
lavad gentx servicer1 10000000000000ulava --chain-id $CHAIN_ID --keyring-backend test || echo failed writing signed gen tx for servicer1
echo "Registering validator"
lavad collect-gentxs
echo "### Initialized new Lava node Successfuly ###"