#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
LOGS_DIR=${__dir}/../../testutil/debugging/logs
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_goerli.json  -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4
lavad tx pairing stake-client "GTH1"   200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Goerli providers
lavad tx pairing stake-provider "GTH1" 2010ulava "127.0.0.1:2121,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" 2000ulava "127.0.0.1:2122,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

screen -d -m -S gth_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2121 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/GTH1_2121.log" && sleep 0.25
screen -S gth_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2122 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/GTH1_2122.log"

screen -d -m -S portals bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3339 GTH1 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"
