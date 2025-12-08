#!/bin/bash 
set -e
echo "Starting init_payment_e2e.sh..."

# Trap errors and show which command failed
trap 'echo "ERROR: Command failed at line $LINENO: $BASH_COMMAND" >&2' ERR

killall lavap 2>/dev/null || true
echo "Killed existing lavap processes"

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh

GASPRICE="0.00002ulava"

# Specs proposal
echo "---- Specs proposal ----"
lavad tx gov submit-legacy-proposal spec-add ./specs/mainnet-1/specs/cosmoswasm.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/cosmossdkv50.json,./specs/testnet-2/specs/lava.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
sleep 8 # need to sleep because plan policies need the specs when setting chain policies verifications

# Plans proposal
echo "---- Plans proposal ----"
wait_next_block
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
sleep 8

STAKE="500000000000ulava"

echo "---- Getting operator address ----"
# Get operator address and ensure it's available
OPERATOR_ADDRESS=$(operator_address)
if [ $? -ne 0 ] || [ -z "$OPERATOR_ADDRESS" ]; then
    echo "ERROR: Failed to get operator address"
    exit 1
fi
echo "Using operator address: $OPERATOR_ADDRESS"

# Lava tendermint/rest providers
echo "---- Staking providers ----"
wait_next_block
wait_next_block
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2261,1" 1 $OPERATOR_ADDRESS -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2262,1" 1 $OPERATOR_ADDRESS -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block

# subscribed clients
echo "---- Creating subscription ----"
wait_next_block
lavad tx subscription buy "DefaultPlan" $(lavad keys show user1 -a) 10 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block

echo "---- Waiting for epoch ----"
sleep_until_next_epoch

# Give lavad a moment to stabilize after all the transactions
echo "Waiting for lavad to stabilize..."
sleep 5

echo "init_payment_e2e.sh completed successfully"
# the end
