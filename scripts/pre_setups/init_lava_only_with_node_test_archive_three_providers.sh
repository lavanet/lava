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

lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/archive.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"
PROVIDER4_LISTENER="127.0.0.1:2223"
PROVIDER5_LISTENER="127.0.0.1:2224"

lavad tx subscription buy ArchivePlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,1,tendermintrpc,rest,grpc,archive" 1 $(operator_address) -y  --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_extension.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block

sleep_until_next_epoch
wait_next_block

echo "[Chaning Epoch Storage Params] submitting param change vote"
lavad tx gov submit-legacy-proposal param-change ./cookbook/param_changes/param_change_epoch_params.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
wait_next_block
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava;

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider --test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
$PROVIDER1_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER1_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$PROVIDER1_LISTENER LAV1 grpc '127.0.0.1:9090' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7766" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider --test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
$PROVIDER2_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER2_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$PROVIDER2_LISTENER LAV1 grpc '127.0.0.1:9090' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ":7756" 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25


screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider config/provider_examples/lava_example_archive.yml --test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_archive.json\
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ":7777"  2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25


wait_next_block

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --debug-relays --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls
