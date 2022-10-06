#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_lava.json,./cookbook/spec_add_ethereum.json,./cookbook/spec_add_osmosis.json,./cookbook/spec_add_fantom.json,./cookbook/spec_add_goerli.json,./cookbook/spec_add_celo.json,./cookbook/spec_add_alfajores.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx gov submit-proposal spec-add ./cookbook/spec_add_arbitrum.json,./cookbook/spec_add_starknet.json,./cookbook/spec_add_aptos.json,./cookbook/spec_add_juno.json,./cookbook/spec_add_cosmoshub.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
lavad tx pairing stake-client "ETH1"   200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "GTH1"   200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS3"   200000ulava 1 -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "FTM250" 200000ulava 1 -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "CELO"   200000ulava 1 -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "LAV1"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS4"   200000ulava 1 -y --from user2 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ALFAJORES" 200000ulava 1 -y --from user3 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ARB1"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ARBN"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "APT1"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "STRK"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "JUN1"   200000ulava 1 -y --from user4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS4"   200000ulava 1 -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS5"   200000ulava 1 -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2000ulava "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2050ulava "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2020ulava "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" 2030ulava "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Goerli providers
lavad tx pairing stake-provider "GTH1" 2010ulava "127.0.0.1:2121,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" 2000ulava "127.0.0.1:2122,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" 2050ulava "127.0.0.1:2123,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" 2020ulava "127.0.0.1:2124,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" 2030ulava "127.0.0.1:2125,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Osmosis providers
lavad tx pairing stake-provider "COS3" 2010ulava "127.0.0.1:2241,tendermintrpc,1 127.0.0.1:2231,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" 2000ulava "127.0.0.1:2242,tendermintrpc,1 127.0.0.1:2232,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" 2050ulava "127.0.0.1:2243,tendermintrpc,1 127.0.0.1:2233,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Fantom providers
lavad tx pairing stake-provider "FTM250" 2010ulava "127.0.0.1:2251,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" 2000ulava "127.0.0.1:2252,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" 2050ulava "127.0.0.1:2253,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" 2020ulava "127.0.0.1:2254,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" 2030ulava "127.0.0.1:2255,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "LAV1" 2010ulava "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2000ulava "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2050ulava "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Juno providers
lavad tx pairing stake-provider "JUN1" 2010ulava "127.0.0.1:2361,tendermintrpc,1 127.0.0.1:2371,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" 2000ulava "127.0.0.1:2362,tendermintrpc,1 127.0.0.1:2372,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" 2050ulava "127.0.0.1:2363,tendermintrpc,1 127.0.0.1:2373,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Osmosis testnet providers
lavad tx pairing stake-provider "COS4" 2010ulava "127.0.0.1:4241,tendermintrpc,1 127.0.0.1:4231,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" 2000ulava "127.0.0.1:4242,tendermintrpc,1 127.0.0.1:4232,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" 2050ulava "127.0.0.1:4243,tendermintrpc,1 127.0.0.1:4233,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Celo providers
lavad tx pairing stake-provider "CELO" 2010ulava "127.0.0.1:5241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" 2000ulava "127.0.0.1:5242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" 2050ulava "127.0.0.1:5243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Celo alfahores testnet providers
lavad tx pairing stake-provider "ALFAJORES" 2010ulava "127.0.0.1:6241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" 2000ulava "127.0.0.1:6242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" 2050ulava "127.0.0.1:6243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Arbitrum mainet providers
lavad tx pairing stake-provider "ARB1" 2010ulava "127.0.0.1:7241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" 2000ulava "127.0.0.1:7242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" 2050ulava "127.0.0.1:7243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Aptos mainet providers
lavad tx pairing stake-provider "APT1" 2010ulava "127.0.0.1:10031,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" 2000ulava "127.0.0.1:10032,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" 2050ulava "127.0.0.1:10033,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Starknet mainet providers
lavad tx pairing stake-provider "STRK" 2010ulava "127.0.0.1:8241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" 2000ulava "127.0.0.1:8242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" 2050ulava "127.0.0.1:8243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# Cosmoshub Providers
lavad tx pairing stake-provider "COS5" 2010ulava "127.0.0.1:2344,tendermintrpc,1 127.0.0.1:2331,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" 2000ulava "127.0.0.1:2342,tendermintrpc,1 127.0.0.1:2332,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" 2050ulava "127.0.0.1:2343,tendermintrpc,1 127.0.0.1:2333,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

. ${__dir}/setup_providers.sh
