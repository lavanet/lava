#!/bin/bash
# init_tron_main.sh - Setup Lava with Tron provider using JSON-RPC interface
# This script tests the REST parameters fix with Tron using REST protocol
# 
# Purpose: Verify that the fix correctly omits params field for no-parameter REST methods
# Expected: Tron-REST accepts requests from Lava provider with correct parameter format
#
# Usage: bash scripts/pre_setups/init_tron_main.sh

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

echo "[Test Setup] Adding Tron spec (with JSON-RPC support)..."
# Use only essential specs to avoid validation issues with uncommitted code changes
# Including: Lava chain spec + Cosmos base specs + Tron
specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/cosmoswasm.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/mainnet-1/specs/tron.json"
echo "[Test Setup] Submitting spec-add proposal (Lava + Tron specs)..."
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

lavad tx pairing stake-provider "TRX" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1,rest" 1 $(operator_address) -y --from servicer1 --provider-moniker "Tron-REST-Provider" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch
wait_next_block

echo "[Test Setup] Starting Tron REST Provider..."
# Tron API Endpoint Selection (tested and verified)
# Default: Tatum Gateway (full API support, no API key, moderate rate limits)
# Preferred: Set TRONGRID_API_KEY for TronGrid (higher rate limits, passed via header)
# Custom: Set TRON_ENDPOINT environment variable

if [ -n "$TRON_ENDPOINT" ]; then
    echo "  Using custom endpoint: $TRON_ENDPOINT"
    screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER TRX rest '$TRON_ENDPOINT' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1_TRON_REST.log" && sleep 2
elif [ -n "$TRONGRID_API_KEY" ]; then
    echo "  Using TronGrid with API key (via header)"
    echo "  Endpoint: https://api.trongrid.io"
    # Create config file with API key header (correct YAML format)
    cat > "$__dir"/../../tron_provider_config.yml <<EOF
endpoints:
  - api-interface: rest
    chain-id: TRX
    network-address:
      address: $PROVIDER1_LISTENER
    node-urls:
      - url: https://api.trongrid.io
        auth-config:
          auth-headers:
            TRON-PRO-API-KEY: "$TRONGRID_API_KEY"
EOF
    screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
tron_provider_config.yml \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1_TRON_REST.log" && sleep 2
else
    TRON_ENDPOINT="https://tron-mainnet.gateway.tatum.io"
    echo "  Using Tatum Gateway (free, but has rate limits)"
    echo "  Endpoint: $TRON_ENDPOINT"
    echo "  Tip: For better rate limits, export TRONGRID_API_KEY=your-key"
    screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER TRX rest '$TRON_ENDPOINT' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1_TRON_REST.log" && sleep 2
fi

wait_next_block

echo "[Test Setup] Starting Tron Consumer..."
screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 TRX rest \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --from user1 --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMERS_TRON.log" && sleep 0.25

echo ""
echo "==================================================================="
echo "✅ Setup complete - Tron REST Test Environment Ready"
echo "==================================================================="
echo ""
echo "Active screens:"
screen -ls
echo ""
echo "Provider logs:"
echo "  tail -f $LOGS_DIR/PROVIDER1_TRON_REST.log"
echo ""
echo "Consumer logs:"
echo "  tail -f $LOGS_DIR/CONSUMERS_TRON.log"
echo ""
echo "Testing the REST parameters fix:"
echo "======================================"
echo ""
echo "The fix is in the REST layer (client.go:301):"
echo "  - Converts nil params → passes nil (not [])"
echo "  - omitempty tag omits 'params' field from JSON"
echo "  - Tron-REST accepts requests without 'params' field"
echo ""
echo "Example test request (no parameters):"
echo "  curl -X POST http://127.0.0.1:3360 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"method\":\"getLatestLedger\",\"id\":1}'"
echo ""
echo "Success indicators:"
echo "  ✅ Request succeeds (no 'params' related errors)"
echo "  ✅ Provider shows successful relay"
echo "  ✅ Tron-REST accepts the request"
echo "  ✅ Logs show REST requests handled correctly"
echo ""
echo "To monitor parameter handling:"
echo "  - Check provider logs for REST requests"
echo "  - Requests should NOT have 'params' field (for no-param methods)"
echo "  - OR should have 'params' exactly as sent by client"
echo "  - Should NOT have 'params':null"
echo ""
echo "To stop everything:"
echo "  killall screen"
echo ""
