#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_45.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json,./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json,./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json,./cookbook/specs/spec_add_avalanche.json,./cookbook/specs/spec_add_fvm.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
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