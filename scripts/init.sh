#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json,./cookbook/spec_add_lava.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

# Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2000ulava "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2050ulava "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2020ulava "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2030ulava "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava providers
lavad tx pairing stake-provider "LAV1" 2010ulava "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2000ulava "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2050ulava "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "LAV1"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch