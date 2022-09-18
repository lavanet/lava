#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice -y --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


sleep 4
lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2000ulava "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

#ETH providers
screen -d -m -S eth1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/ETH1_2221.log"
# screen -S eth1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/ETH1_2222.log"

sleep 1
screen -d -m -S portals zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3333.log"
screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3334 ETH1 jsonrpc --from user2 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"
screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 ETH1 jsonrpc --from user3 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"
screen -S portals -X screen -t win12 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3336 ETH1 jsonrpc --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"

sleep 1
screen -ls 