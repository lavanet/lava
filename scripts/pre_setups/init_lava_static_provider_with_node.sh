#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe

echo "[Test Setup] installing all binaries"
make install-all 

echo "[Test Setup] setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Test Setup] sleeping 20 seconds for node to finish setup (if its not enough increase timeout)"
sleep 5
wait_for_lava_node_to_start

GASPRICE="0.00002ulava"
specs=$(get_all_specs)
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

# Plans proposal
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
# static configuration
PROVIDER4_LISTENER="127.0.0.1:2220"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
# lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1"  1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level trace --from servicer1 --chain-id lava --metrics-listen-address ":7776" --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

wait_next_block

screen -d -m -S provider4 bash -c "source ~/.bashrc; lavap rpcprovider provider_examples/lava_example.yml\
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer4 --static-providers --chain-id lava --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/PROVIDER4.log" && sleep 0.25

screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider provider_examples/lava_example2.yml\
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --static-providers --chain-id lava --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

<<<<<<< HEAD:scripts/pre_setups/init_lava_static_provider_with_node.sh
screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcsmartrouter consumer_examples/lava_consumer_static_peers.yml \
<<<<<<< HEAD
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --allow-insecure-provider-dialing --metrics-listen-address ":7779" --enable-provider-optimizer-auto-adjustment-of-tiers 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25
=======
screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/lava_consumer_static_peers.yml \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" --use-lava-over-lava-backup=false 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25
>>>>>>> 842864bd0 (test: Update provider optimizer tests for weighted selection logic):scripts/pre_setups/init_lava_static_provider.sh
=======
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25
>>>>>>> 1b80781ea (Refactor provider optimizer initialization to remove unused parameter)

echo "--- setting up screens done ---"
screen -ls
