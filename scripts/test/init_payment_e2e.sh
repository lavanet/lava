#!/bin/bash 
killall lavap
set -e

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh

GASPRICE="0.00002ulava"

# Specs proposal
lavad tx gov submit-legacy-proposal spec-add ./specs/mainnet-1/specs/cosmoswasm.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/cosmossdkv50.json,./specs/testnet-2/specs/lava.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov deposit 1 100ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6 # need to sleep because plan policies need the specs when setting chain policies verifications

# Plans proposal
wait_next_block
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov deposit 2 100ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6

STAKE="500000000000ulava"

# Lava tendermint/rest providers
wait_next_block
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2261,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2262,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# subscribed clients
lavad tx subscription buy "DefaultPlan" $(lavad keys show user1 -a) 10 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
sleep_until_next_epoch

# the end
