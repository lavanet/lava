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
echo "[Test Setup] waiting blocks to start progressing"
sleep 3
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

PROVIDER1_LISTENER="127.0.0.1:2220"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER EVMOS rest '$EVMOS_REST' \
$PROVIDER1_LISTENER EVMOS tendermintrpc '$EVMOS_RPC,$EVMOS_RPC' \
$PROVIDER1_LISTENER EVMOS grpc '$EVMOS_GRPC' \
$PROVIDER1_LISTENER EVMOS jsonrpc '$EVMOS_JRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7776" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25


screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25
sleep 2;

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 EVMOS rest 127.0.0.1:3361 EVMOS tendermintrpc 127.0.0.1:3362 EVMOS grpc 127.0.0.1:3363 EVMOS jsonrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --cache-be 127.0.0.1:20100 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

# sleep 30
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_extension.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# sleep 30
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
