#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
LOGS_DIR=${__dir}/../../testutil/debugging/logs
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_lava.json,./cookbook/spec_add_ethereum.json,./cookbook/spec_add_osmosis.json,./cookbook/spec_add_fantom.json  -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4 
STAKE="500000000000ulava"
lavad tx pairing stake-client "COS4" $STAKE 1 -y --from user4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "COS4" $STAKE "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1 127.0.0.1:2281,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

# Lava providers
screen -d -m -S cos4_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2271 $OSMO_TEST_REST COS4 rest --from servicer1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/COS4_2271.log"; sleep 0.3
# screen -S cos4_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2261 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/COS4_2261.log"
screen -d -m -S portals bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3340 COS4 rest 127.0.0.1:3341 COS4 tendermintrpc --from user4 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug | tee $LOGS_DIR/COS4_tendermint_portal.log"; sleep 0.3

lavad server 127.0.0.1 2261 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --tendermint-http-endpoint $OSMO_TEST_RPC_HTTP
# Lava Over Lava ETH

sleep 3 # wait for the portal to start.

# lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user1 --node "http://127.0.0.1:3341/1"

# lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --node "http://127.0.0.1:3341/1"

# sleep_until_next_epoch

# screen -d -m -S eth1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 --node \"http://127.0.0.1:3341/1\" 2>&1 | tee $LOGS_DIR/ETH1_2221.log"

# screen -d -m -S eth1_portals bash -c "source ~/.bashrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1 --node \"http://127.0.0.1:3341/1\" 2>&1 | tee $LOGS_DIR/PORTAL_3333.log"

screen -ls