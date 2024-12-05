#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
LOGS_DIR=${__dir}/../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.00002ulava"

# ,./cookbook/specs/mantle.json
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/ibc.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_45.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,./cookbook/specs/lava.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_count_blocks 2
echo "submitted first proposal"
echo "latest vote2: $(latest_vote)"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
echo "voted on first proposal"
sleep 4
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 2
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"
PROVIDER4_LISTENER="127.0.0.1:2224"

sleep 4
lavad tx  subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# MANTLE
CHAINS="ETH1,LAV1"
# stake providers on all chains
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER4_LISTENER,1" 1 $(operator_address) -y --from servicer4 --provider-moniker "servicer4" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER1_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --chain-id=lava --metrics-listen-address ":7780" --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log"

screen -d -m -S provider2 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER2_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER2_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER2_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --chain-id=lava --metrics-listen-address ":7780" --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/PROVIDER2.log"

screen -d -m -S provider3 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER3_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER3_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER3_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --chain-id=lava --metrics-listen-address ":7780" --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/PROVIDER3.log"

screen -d -m -S provider4 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER4_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER4_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER4_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER4_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --chain-id=lava --metrics-listen-address ":7780" --geolocation 1 --log_level debug --from servicer4 2>&1 | tee $LOGS_DIR/PROVIDER4.log"

# Setup Consumer
screen -d -m -S portals bash -c "source ~/.bashrc; lava-protocol rpcconsumer \
127.0.0.1:3333 ETH1 jsonrpc \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --metrics-listen-address ":7779" --geolocation 1 --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMER.log"


# need to wait 8 epochs for the provider to be jail eligible
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch
sleep_until_next_epoch


screen -d -m -S testing bash -c "source ~/.bashrc; lavad test rpcconsumer http://127.0.0.1:3333 ETH1 jsonrpc http://127.0.0.1:3360 LAV1 rest http://127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc --chain-id lava 2>&1 | tee $LOGS_DIR/TESTING.log"

echo "running some relays, before terminating provider4"

sleep_until_next_epoch
sleep_until_next_epoch

screen -S provider4 -X quit

lavad test events 0 --from servicer4