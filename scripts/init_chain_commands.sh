#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 4 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
lavad tx gov submit-proposal plans-add ./cookbook/specs/plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 5 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"

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
lavad tx pairing stake-client "BSC"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "AXELAR"  $CLIENTSTAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE



# Ethereum providers
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Goerli providers
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "GTH1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Fantom providers
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "FTM250" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Celo providers
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CELO" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Celo alfahores testnet providers
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ALFAJORES" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Arbitrum mainet providers
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ARB1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Aptos mainet providers
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,rest,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,rest,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "APT1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,rest,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

#Starknet mainet providers
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "STRK" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Polygon Providers
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "POLYGON1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Optimism Providers
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "OPTM" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Base Providers
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BASET" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Sui Providers
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SUIT" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# SOLANA Providers
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "SOLANA" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# BSC Providers
lavad tx pairing stake-provider "BSC" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BSC" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "BSC" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


# Cosmos Chains:

# Osmosis providers
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS3" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Juno providers
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "JUN1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Osmosis testnet providers
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS4" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Cosmoshub Providers
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "COS5" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Evmos providers
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1 $PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1 $PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "EVMOS" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1 $PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# canto Providers
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "$PROVIDER1_LISTENER,jsonrpc,1 $PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "$PROVIDER2_LISTENER,jsonrpc,1 $PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "CANTO" $PROVIDERSTAKE "$PROVIDER3_LISTENER,jsonrpc,1 $PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# axelar Providers
lavad tx pairing stake-provider "AXELAR" $PROVIDERSTAKE "$PROVIDER1_LISTENER,tendermintrpc,1 $PROVIDER1_LISTENER,rest,1 $PROVIDER1_LISTENER,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "AXELAR" $PROVIDERSTAKE "$PROVIDER2_LISTENER,tendermintrpc,1 $PROVIDER2_LISTENER,rest,1 $PROVIDER2_LISTENER,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "AXELAR" $PROVIDERSTAKE "$PROVIDER3_LISTENER,tendermintrpc,1 $PROVIDER3_LISTENER,rest,1 $PROVIDER3_LISTENER,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

. ${__dir}/setup_providers.sh
