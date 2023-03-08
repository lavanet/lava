#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_cosmossdk.json,./cookbook/spec_add_ethereum.json,./cookbook/spec_add_cosmoshub.json,./cookbook/spec_add_lava.json,./cookbook/spec_add_osmosis.json,./cookbook/spec_add_fantom.json,./cookbook/spec_add_celo.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

lavad tx gov submit-proposal spec-add ./cookbook/spec_add_optimism.json,./cookbook/spec_add_arbitrum.json,./cookbook/spec_add_starknet.json,./cookbook/spec_add_aptos.json,./cookbook/spec_add_juno.json,./cookbook/spec_add_polygon.json,./cookbook/spec_add_evmos.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE



sleep 4
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_base.json,./cookbook/spec_add_canto.json,./cookbook/spec_add_sui.json,./cookbook/spec_add_solana.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx gov submit-proposal plans-add ./cookbook/plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 4 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

sleep 4
lavad tx pairing stake-client "ETH1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "GTH1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS3"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "FTM250" $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "CELO"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "LAV1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS4"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ALFAJORES" $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ARB1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "ARBN"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "APT1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "STRK"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto"  --gas-prices $GASPRICE
lavad tx pairing stake-client "JUN1"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "COS5"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "POLYGON1" $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "EVMOS"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "OPTM"   $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "BASET"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "CANTO"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "SUIT"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "SOLANA"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


# Ethereum providers
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Goerli providers
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "127.0.0.1:2121,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "127.0.0.1:2122,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "127.0.0.1:2123,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "127.0.0.1:2124,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "127.0.0.1:2125,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Fantom providers
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "127.0.0.1:2251,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "127.0.0.1:2252,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "127.0.0.1:2253,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "127.0.0.1:2254,jsonrpc,1" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "127.0.0.1:2255,jsonrpc,1" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Celo providers
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "127.0.0.1:5241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "127.0.0.1:5242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "127.0.0.1:5243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Celo alfahores testnet providers
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "127.0.0.1:6241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "127.0.0.1:6242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "127.0.0.1:6243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Arbitrum mainet providers
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "127.0.0.1:7241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "127.0.0.1:7242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "127.0.0.1:7243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Aptos mainet providers
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "127.0.0.1:10031,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "127.0.0.1:10032,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "127.0.0.1:10033,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Starknet mainet providers
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "127.0.0.1:8241,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "127.0.0.1:8242,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "127.0.0.1:8243,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Polygon Providers
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "127.0.0.1:4344,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "127.0.0.1:4345,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "127.0.0.1:4346,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Optimism Providers
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "127.0.0.1:6003,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "127.0.0.1:6004,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "127.0.0.1:6005,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Base Providers
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "127.0.0.1:6000,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "127.0.0.1:6001,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "127.0.0.1:6002,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Sui Providers
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "127.0.0.1:6500,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "127.0.0.1:6501,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "127.0.0.1:6502,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# SOLANA Providers
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "127.0.0.1:6510,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "127.0.0.1:6511,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "127.0.0.1:6512,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Cosmos Chains:

# Osmosis providers
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "127.0.0.1:2241,tendermintrpc,1 127.0.0.1:2231,rest,1 127.0.0.1:2234,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "127.0.0.1:2242,tendermintrpc,1 127.0.0.1:2232,rest,1 127.0.0.1:2235,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "127.0.0.1:2243,tendermintrpc,1 127.0.0.1:2233,rest,1 127.0.0.1:2236,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1 127.0.0.1:2274,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1 127.0.0.1:2275,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1 127.0.0.1:2276,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Juno providers
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "127.0.0.1:2361,tendermintrpc,1 127.0.0.1:2371,rest,1 127.0.0.1:2374,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "127.0.0.1:2362,tendermintrpc,1 127.0.0.1:2372,rest,1 127.0.0.1:2375,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "127.0.0.1:2363,tendermintrpc,1 127.0.0.1:2373,rest,1 127.0.0.1:2376,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Osmosis testnet providers
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "127.0.0.1:4241,tendermintrpc,1 127.0.0.1:4231,rest,1 127.0.0.1:4234,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "127.0.0.1:4242,tendermintrpc,1 127.0.0.1:4232,rest,1 127.0.0.1:4235,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "127.0.0.1:4243,tendermintrpc,1 127.0.0.1:4233,rest,1 127.0.0.1:4236,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Cosmoshub Providers
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "127.0.0.1:2344,tendermintrpc,1 127.0.0.1:2331,rest,1 127.0.0.1:2334,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "127.0.0.1:2342,tendermintrpc,1 127.0.0.1:2332,rest,1 127.0.0.1:2335,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "127.0.0.1:2343,tendermintrpc,1 127.0.0.1:2333,rest,1 127.0.0.1:2336,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Evmos providers
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "127.0.0.1:4347,jsonrpc,1 127.0.0.1:4348,tendermintrpc,1 127.0.0.1:4349,rest,1 127.0.0.1:4350,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "127.0.0.1:4351,jsonrpc,1 127.0.0.1:4352,tendermintrpc,1 127.0.0.1:4353,rest,1 127.0.0.1:4354,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "127.0.0.1:4355,jsonrpc,1 127.0.0.1:4356,tendermintrpc,1 127.0.0.1:4357,rest,1 127.0.0.1:4358,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# canto Providers
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "127.0.0.1:6006,jsonrpc,1 127.0.0.1:6009,tendermintrpc,1 127.0.0.1:6012,rest,1 127.0.0.1:6015,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "127.0.0.1:6007,jsonrpc,1 127.0.0.1:6010,tendermintrpc,1 127.0.0.1:6013,rest,1 127.0.0.1:6016,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "127.0.0.1:6008,jsonrpc,1 127.0.0.1:6011,tendermintrpc,1 127.0.0.1:6014,rest,1 127.0.0.1:6017,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

. ${__dir}/setup_providers.sh
