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
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/ibc.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_45.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,./cookbook/specs/cosmoshub.json,./cookbook/specs/lava.json,./cookbook/specs/osmosis.json,./cookbook/specs/fantom.json,./cookbook/specs/celo.json,./cookbook/specs/optimism.json,./cookbook/specs/arbitrum.json,./cookbook/specs/starknet.json,./cookbook/specs/aptos.json,./cookbook/specs/juno.json,./cookbook/specs/polygon.json,./cookbook/specs/evmos.json,./cookbook/specs/base.json,./cookbook/specs/canto.json,./cookbook/specs/sui.json,./cookbook/specs/solana.json,./cookbook/specs/bsc.json,./cookbook/specs/axelar.json,./cookbook/specs/avalanche.json,./cookbook/specs/fvm.json,./cookbook/specs/celestia.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
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

PROVIDER1_LISTENER="127.0.0.1:2220"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "AXELAR" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER CELESTIATM rest '$CELESTIA_REST' \
$PROVIDER1_LISTENER CELESTIATM tendermintrpc '$CELESTIA_RPC,$CELESTIA_RPC' \
$PROVIDER1_LISTENER CELESTIATM grpc '$CELESTIA_GRPC' \
$PROVIDER1_LISTENER CELESTIATM jsonrpc '$CELESTIA_JSONRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7776" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 CELESTIATM rest 127.0.0.1:3361 CELESTIATM tendermintrpc 127.0.0.1:3362 CELESTIATM grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls