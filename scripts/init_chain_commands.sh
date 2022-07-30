#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh

# lavad tx gov submit-proposal spec-add ./cookbook/spec_add_terra.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov vote 1 yes -y --from alice
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_osmosis.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov vote 2 yes -y --from alice
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_fantom.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov vote 3 yes -y --from alice
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_lava.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov vote 4 yes -y --from alice
sleep 4
lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user1
lavad tx pairing stake-client "COS3" 200000ulava 1 -y --from user2
lavad tx pairing stake-client "FTM250" 200000ulava 1 -y --from user3
lavad tx pairing stake-client "LAV1" 200000ulava 1 -y --from user4

# Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1
lavad tx pairing stake-provider "ETH1" 2000ulava "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2
lavad tx pairing stake-provider "ETH1" 2050ulava "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3
lavad tx pairing stake-provider "ETH1" 2020ulava "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4
lavad tx pairing stake-provider "ETH1" 2030ulava "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5

# Osmosis providers
lavad tx pairing stake-provider "COS3" 2010ulava "127.0.0.1:2241,tendermintrpc,1 127.0.0.1:2231,rest,1" 1 -y --from servicer1
lavad tx pairing stake-provider "COS3" 2000ulava "127.0.0.1:2242,tendermintrpc,1 127.0.0.1:2232,rest,1" 1 -y --from servicer2
lavad tx pairing stake-provider "COS3" 2050ulava "127.0.0.1:2243,tendermintrpc,1 127.0.0.1:2233,rest,1" 1 -y --from servicer3
# lavad tx pairing stake-provider "COS1" 2020ulava "127.0.0.1:2244,tendermintrpc,1 127.0.0.1:2234,rest,1" 1 -y --from servicer4
# lavad tx pairing stake-provider "COS1" 2030ulava "127.0.0.1:2245,tendermintrpc,1 127.0.0.1:2235,rest,1" 1 -y --from servicer5

# Fantom providers
lavad tx pairing stake-provider "FTM250" 2010ulava "127.0.0.1:2251,jsonrpc,1" 1 -y --from servicer1
lavad tx pairing stake-provider "FTM250" 2000ulava "127.0.0.1:2252,jsonrpc,1" 1 -y --from servicer2
lavad tx pairing stake-provider "FTM250" 2050ulava "127.0.0.1:2253,jsonrpc,1" 1 -y --from servicer3
lavad tx pairing stake-provider "FTM250" 2020ulava "127.0.0.1:2254,jsonrpc,1" 1 -y --from servicer4
lavad tx pairing stake-provider "FTM250" 2030ulava "127.0.0.1:2255,jsonrpc,1" 1 -y --from servicer5

# Lava Providers
lavad tx pairing stake-provider "LAV1" 2010ulava "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1" 1 -y --from servicer1
lavad tx pairing stake-provider "LAV1" 2000ulava "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1" 1 -y --from servicer2
lavad tx pairing stake-provider "LAV1" 2050ulava "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1" 1 -y --from servicer3

echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"
echo "---------------Setup Providers------------------"
killall screen

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

# Eth providers
screen -d -m -S eth1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1"
screen -S eth1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3"
screen -S eth1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4"
screen -S eth1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5"

# Osmosis providers
screen -d -m -S cos3_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest --from servicer1"
screen -S cos3_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest --from servicer2"
screen -S cos3_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest --from servicer3"
screen -S cos3_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc --from servicer1"
screen -S cos3_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc --from servicer2"
screen -S cos3_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc --from servicer3"

# FTM providers
# screen -d -m -S ftm250_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2251 $FTM_RPC_HTTPS FTM250 jsonrpc --from servicer1"
# screen -S ftm250_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2252 $FTM_RPC_HTTPS FTM250 jsonrpc --from servicer2"
# screen -S ftm250_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2253 $FTM_RPC_HTTPS FTM250 jsonrpc --from servicer3"
# screen -S ftm250_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2254 $FTM_RPC_HTTPS FTM250 jsonrpc --from servicer4"
# screen -S ftm250_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2255 $FTM_RPC_HTTPS FTM250 jsonrpc --from servicer5"

# Lava providers
screen -d -m -S lav1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2271 $LAVA_REST LAV1 rest --from servicer1"
screen -S lav1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2272 $LAVA_REST LAV1 rest --from servicer2"
screen -S lav1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2273 $LAVA_REST LAV1 rest --from servicer3"
screen -S lav1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2261 $LAVA_RPC LAV1 tendermintrpc --from servicer1"
screen -S lav1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2262 $LAVA_RPC LAV1 tendermintrpc --from servicer2"
screen -S lav1_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2263 $LAVA_RPC LAV1 tendermintrpc --from servicer3"

screen -d -m -S portals zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3334 COS3 rest --from user2"
screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2"
# screen -S portals -X screen -t win12 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3336 FTM250 jsonrpc --from user3"
screen -S portals -X screen -t win13 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3337 LAV1 rest --from user4"
screen -S portals -X screen -t win14 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3338 LAV1 tendermintrpc --from user4"

echo "--- setting up screens done ---"
screen -ls
