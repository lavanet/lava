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
PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"
PROVIDER4_LISTENER="127.0.0.1:2224"
PROVIDER5_LISTENER="127.0.0.1:2225"
if [ $# -eq 0 ]; then
    lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/ibc.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_45.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,./cookbook/specs/cosmoshub.json,./cookbook/specs/lava.json,./cookbook/specs/osmosis.json,./cookbook/specs/fantom.json,./cookbook/specs/celo.json,./cookbook/specs/optimism.json,./cookbook/specs/arbitrum.json,./cookbook/specs/starknet.json,./cookbook/specs/aptos.json,./cookbook/specs/juno.json,./cookbook/specs/polygon.json,./cookbook/specs/evmos.json,./cookbook/specs/base.json,./cookbook/specs/canto.json,./cookbook/specs/sui.json,./cookbook/specs/solana.json,./cookbook/specs/bsc.json,./cookbook/specs/axelar.json,./cookbook/specs/avalanche.json,./cookbook/specs/fvm.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
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

    lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    wait_next_block

    lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_extension.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

    lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --delegate-commission 50 
    lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,2" 2 $(operator_address) -y --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --delegate-commission 50 
    lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,2" 2 $(operator_address) -y --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --delegate-commission 50 
    lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER4_LISTENER,2,archive" 2 $(operator_address) -y --from servicer4 --provider-moniker "servicer4" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --delegate-commission 50 
    lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER5_LISTENER,1,archive" 1 $(operator_address) -y --from servicer5 --provider-moniker "servicer5" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --delegate-commission 50 

    sleep_until_next_epoch
    lavad q pairing effective-policy ETH1 $(lavad keys show user1 -a)
fi


i=1
screen -d -m -S provider$i bash -c "source ~/.bashrc; lavap rpcprovider \
127.0.0.1:222$i ETH1 jsonrpc '$ETH_RPC_WS' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer$i --chain-id lava 2>&1 | tee $LOGS_DIR/PROVIDER$i.log" && sleep 0.25

for i in {2..4}; do
  screen -d -m -S provider$i bash -c "source ~/.bashrc; lavap rpcprovider \
  127.0.0.1:222$i ETH1 jsonrpc '$ETH_RPC_WS' \
  $EXTRA_PROVIDER_FLAGS --geolocation 2 --log_level debug --from servicer$i --chain-id lava 2>&1 | tee $LOGS_DIR/PROVIDER$i.log" && sleep 0.25
done

i=5
screen -d -m -S provider$i bash -c "source ~/.bashrc; lavap rpcprovider \
127.0.0.1:222$i ETH1 jsonrpc '$ETH_RPC_WS' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer$i --chain-id lava 2>&1 | tee $LOGS_DIR/PROVIDER$i.log" && sleep 0.25

screen -d -m -S portals bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/ethereum_example.yml\
$EXTRA_PORTAL_FLAGS --cache-be "127.0.0.1:7778" --geolocation 1 --debug-relays --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25
echo "--- setting up screens done ---"
screen -ls

sleep 5
lavad test rpcconsumer http://127.0.0.1:3333 ETH1 jsonrpc --chain-id lava
