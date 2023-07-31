#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

GASPRICE="0.000000001ulava"
NODE="http://127.0.0.1:3340/1"
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_ethereum.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE

lavad tx gov submit-proposal plans-add ./cookbook/plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4

STAKE="500000000000ulava"
# Goerli providers
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2121,1" 1 -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2122,1" 1 -y --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2123,1" 1 -y --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2124,1" 1 -y --from servicer4 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2125,1" 1 -y --from servicer5 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE

lavad tx  subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch