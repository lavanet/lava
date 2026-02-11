#!/bin/bash
# init_stellar_main.sh - Setup Lava with Stellar provider using JSON-RPC interface
# This script tests the JSON-RPC parameters fix with Stellar using JSON-RPC protocol
# 
# Purpose: Verify that the fix correctly omits params field for no-parameter JSON-RPC methods
# Expected: Stellar-RPC accepts requests from Lava provider with correct parameter format
#
# Usage: bash scripts/pre_setups/init_stellar_main.sh

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm -f $LOGS_DIR/*.log

killall screen
screen -wipe

echo "[Test Setup] Installing binaries..."
make install-all 

echo "[Test Setup] Setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Test Setup] Waiting for node to finish setup..."
sleep 5
wait_for_lava_node_to_start
echo "[Test Setup] Lava node started successfully"

GASPRICE="0.00002ulava"

echo "[Test Setup] Adding Stellar spec (with JSON-RPC support)..."
# Use only essential specs to avoid validation issues with uncommitted code changes
# Including: Lava chain spec + Cosmos base specs + Stellar
specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/cosmoswasm.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/mainnet-1/specs/stellar.json,specs/mainnet-1/specs/ripple.json"
echo "[Test Setup] Submitting spec-add proposal (Lava + Stellar specs)..."
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to submit spec-add proposal"
    exit 1
fi
echo "[Test Setup] Spec proposal submitted successfully, waiting for blocks..."
wait_next_block
wait_next_block
echo "[Test Setup] Voting on spec proposal #1..."
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to vote on spec proposal"
    exit 1
fi
wait_next_block

echo "[Test Setup] Adding default plan..."
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to submit plans-add proposal"
    exit 1
fi
wait_next_block
wait_next_block
echo "[Test Setup] Voting on plan proposal #2..."
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
if [ $? -ne 0 ]; then
    echo "ERROR: Failed to vote on plan proposal"
    exit 1
fi

sleep 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2220"

echo "[Test Setup] Setting up subscription and provider stake..."
lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block

lavad tx pairing stake-provider "XRP" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1,jsonrpc" 1 $(operator_address) -y --from servicer1 --provider-moniker "Ripple-JSON-RPC-Provider" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch
wait_next_block

echo "[Test Setup] Starting Ripple JSON-RPC Provider..."
echo "  Using public Ripple JSON-RPC endpoint: https://ripple.drpc.org"
echo "  Testing JSON-RPC interface (exercises the parameters fix)"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER XRP jsonrpc 'https://s2.ripple.com:51234/,wss://s1.ripple.com/' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1_RIPPLE_JSONRPC.log" && sleep 2

wait_next_block

echo "[Test Setup] Starting Ripple Consumer..."
screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 XRP jsonrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --from user1 --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMERS_RIPPLE.log" && sleep 0.25

echo ""
echo "==================================================================="
echo "✅ Setup complete - Ripple JSON-RPC Test Environment Ready"
echo "==================================================================="
echo ""
echo "Active screens:"
screen -ls
echo ""
echo "Provider logs:"
echo "  tail -f $LOGS_DIR/PROVIDER1_RIPPLE_JSONRPC.log"
echo ""
echo "Consumer logs:"
echo "  tail -f $LOGS_DIR/CONSUMERS_RIPPLE.log"
echo ""
echo "Testing the JSON-RPC parameters fix:"
echo "======================================"
echo ""
echo "The fix is in the JSON-RPC layer (client.go:301):"
echo "  - Converts nil params → passes nil (not [])"
echo "  - omitempty tag omits 'params' field from JSON"
echo "  - Ripple-RPC accepts requests without 'params' field"
echo ""
echo "Example test request (no parameters):"
echo "  curl -X POST http://127.0.0.1:3360 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"method\":\"getLatestLedger\",\"id\":1}'"
echo ""
echo "Success indicators:"
echo "  ✅ Request succeeds (no 'params' related errors)"
echo "  ✅ Provider shows successful relay"
echo "  ✅ Ripple-RPC accepts the request"
echo "  ✅ Logs show JSON-RPC requests handled correctly"
echo ""
echo "To monitor parameter handling:"
echo "  - Check provider logs for JSON-RPC requests"
echo "  - Requests should NOT have 'params' field (for no-param methods)"
echo "  - OR should have 'params' exactly as sent by client"
echo "  - Should NOT have 'params':null"
echo ""
echo "To stop everything:"
echo "  killall screen"
echo ""
