#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@";
echo "Either use the lava cache header in requests";
echo "Or change the pruning configuration before running";
echo "If you did dont commit the changes.";
echo "@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@";
sleep 3;

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
lavad tx gov submit-legacy-proposal spec-add ./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmoswasm.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/cosmossdk_45.json,./specs/mainnet-1/specs/cosmossdk_full.json,./specs/mainnet-1/specs/ethermint.json,./specs/mainnet-1/specs/ethereum.json,./specs/mainnet-1/specs/cosmoshub.json,./specs/mainnet-1/specs/lava.json,./specs/mainnet-1/specs/osmosis.json,./specs/mainnet-1/specs/fantom.json,./specs/mainnet-1/specs/celo.json,./specs/mainnet-1/specs/optimism.json,./specs/mainnet-1/specs/arbitrum.json,./specs/mainnet-1/specs/starknet.json,./specs/mainnet-1/specs/aptos.json,./specs/mainnet-1/specs/juno.json,./specs/mainnet-1/specs/polygon.json,./specs/mainnet-1/specs/evmos.json,./specs/mainnet-1/specs/base.json,./specs/mainnet-1/specs/canto.json,./specs/mainnet-1/specs/sui.json,./specs/mainnet-1/specs/solana.json,./specs/mainnet-1/specs/bsc.json,./specs/mainnet-1/specs/axelar.json,./specs/mainnet-1/specs/avalanche.json,./specs/mainnet-1/specs/fvm.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
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
PROVIDER2_LISTENER="127.0.0.1:2220"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y  --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,1,tendermintrpc,rest,grpc,archive" 1 $(operator_address) -y  --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_extension.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
sleep_until_next_epoch

screen -d -m -S cache_consumer bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE_CONSUMER.log" && sleep 0.25
sleep 2;
screen -d -m -S cache_provider bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20101 --metrics_address 0.0.0.0:20201 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE_PROVIDER.log" && sleep 0.25
sleep 2;

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --cache-be 127.0.0.1:20101 --metrics-listen-address ":7776" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider config/provider_examples/lava_example_archive.yml \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ":7777" --cache-be 127.0.0.1:20101 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --shared-state --chain-id lava --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

screen -d -m -S consumer2 bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3363 LAV1 rest 127.0.0.1:3364 LAV1 tendermintrpc 127.0.0.1:3365 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --shared-state --chain-id lava --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS2.log" && sleep 0.25


echo "--- setting up screens done ---"
screen -ls

