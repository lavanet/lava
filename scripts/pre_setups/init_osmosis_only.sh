#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe
GASPRICE="0.00002ulava"
lavad tx gov submit-legacy-proposal spec-add ./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmoswasm.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/cosmossdkv45.json,./specs/mainnet-1/specs/cosmossdkv50.json,./specs/mainnet-1/specs/ethermint.json,./specs/mainnet-1/specs/ethereum.json,./specs/mainnet-1/specs/cosmoshub.json,./specs/mainnet-1/specs/lava.json,./specs/mainnet-1/specs/osmosis.json,./specs/mainnet-1/specs/fantom.json,./specs/mainnet-1/specs/celo.json,./specs/mainnet-1/specs/optimism.json,./specs/mainnet-1/specs/arbitrum.json,./specs/mainnet-1/specs/starknet.json,./specs/mainnet-1/specs/aptos.json,./specs/mainnet-1/specs/juno.json,./specs/mainnet-1/specs/polygon.json,./specs/mainnet-1/specs/evmos.json,./specs/mainnet-1/specs/base.json,./specs/mainnet-1/specs/canto.json,./specs/mainnet-1/specs/sui.json,./specs/mainnet-1/specs/solana.json,./specs/mainnet-1/specs/bsc.json,./specs/mainnet-1/specs/axelar.json,./specs/mainnet-1/specs/avalanche.json,./specs/mainnet-1/specs/fvm.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4


CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx pairing stake-provider "OSMOSIS" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER OSMOSIS rest '$OSMO_REST' \
$PROVIDER1_LISTENER OSMOSIS tendermintrpc '$OSMO_RPC,$OSMO_RPC' \
$PROVIDER1_LISTENER OSMOSIS grpc '$OSMO_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 OSMOSIS rest 127.0.0.1:3361 OSMOSIS tendermintrpc 127.0.0.1:3362 OSMOSIS grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls