#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

GASPRICE="0.000000001ulava"
NODE="http://127.0.0.1:3340/1"
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_ethereum.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
sleep 4

STAKE="500000000000ulava"
# Goerli providers
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2121,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2122,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2123,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2124,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE
lavad tx pairing stake-provider "GTH1" $STAKE "127.0.0.1:2125,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE

lavad tx pairing stake-client "GTH1" $STAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE --node $NODE

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch