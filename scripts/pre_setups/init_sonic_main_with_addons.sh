#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe

echo "[Test Setup] installing all binaries"
make install-all 


echo "[Test Setup] setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Test Setup] sleeping 20 seconds for node to finish setup (if its not enough increase timeout)"
sleep 5
wait_for_lava_node_to_start

specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/cosmoswasm.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/mainnet-1/specs/ethereum.json,specs/mainnet-1/specs/sonic.json"
GASPRICE="0.00002ulava"
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block

lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
# Stake provider with trace add-on only (public RPC endpoints don't support debug)
lavad tx pairing stake-provider "SONIC" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1,trace" 1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch
wait_next_block

# Set project policy to allow trace add-on (and potentially debug if available)
echo "Setting project policy to allow trace and debug add-ons for SONIC"
lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_sonic_with_addons.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
sleep_until_next_epoch

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
./config/provider_examples/sonic_provider_with_addons.yml \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address \":7776\" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

wait_next_block

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 SONIC jsonrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "=============================================="
echo "Waiting for services to stabilize..."
echo "=============================================="
sleep 10

echo ""
echo "=============================================="
echo "Verifying Provider Configuration"
echo "=============================================="
echo "Checking provider logs for add-on configuration..."
if grep -q "Addons:trace" $LOGS_DIR/PROVIDER1.log; then
    echo "✅ Provider correctly configured with trace add-on"
    if grep -q "Addons:debug" $LOGS_DIR/PROVIDER1.log; then
        echo "✅ Provider also has debug add-on available"
    else
        echo "ℹ️  Note: Debug add-on not available (requires private RPC node)"
    fi
else
    echo "⚠️  WARNING: Provider add-ons not detected in logs yet"
    echo "    This may cause add-on API calls to fail"
    echo "    Check PROVIDER1.log for 'Addons:' configuration"
fi

echo ""
echo "Checking if provider staked with add-ons..."
lavad query pairing account-info --from servicer1 2>/dev/null | grep -A 20 "SONIC" || echo "Provider info query failed"

echo ""
echo "=============================================="
echo "Testing Standard APIs (inherited from ETH1)"
echo "=============================================="

echo ""
echo "Test 1: eth_blockNumber"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq

echo ""
echo "Test 2: eth_chainId"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' | jq

echo ""
echo "=============================================="
echo "Testing TRACE Add-on (Sonic-specific + ETH1)"
echo "=============================================="

echo ""
echo "Test 3: trace_block (Sonic-specific trace API)"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"trace_block","params":["latest"],"id":1}' | jq -r '.error // .result | if type == "string" then . elif type == "array" then "✅ SUCCESS: Returned \(length) traces" else . end'

echo ""
echo "Test 4: trace_transaction (Sonic-specific trace API)"
echo "(Note: This will fail if no recent transactions exist, which is expected)"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"trace_transaction","params":["0x1234567890123456789012345678901234567890123456789012345678901234"],"id":1}' | jq -r '.error // .result // "No transaction found (expected)"'

echo ""
echo "=============================================="
echo "Testing DEBUG Add-on (inherited from ETH1)"
echo "=============================================="
echo "ℹ️  Note: Debug add-on requires dedicated RPC node (not available on public endpoints)"
echo ""

echo ""
echo "Test 5: debug_traceBlockByNumber (ETH1 debug API) - Expected to fail with public RPC"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"debug_traceBlockByNumber","params":["latest",{"tracer":"callTracer"}],"id":1}' | jq -r '.error // .result | if type == "string" then if (. | contains("does not exist")) or (. | contains("not available")) then "⚠️  Expected: Debug APIs not available on public RPC" else . end elif type == "object" then "✅ SUCCESS: Debug trace returned" else . end'

echo ""
echo "Test 6: debug_getRawHeader (ETH1 debug API) - Expected to fail with public RPC"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"debug_getRawHeader","params":["latest"],"id":1}' | jq -r '.error // .result | if type == "string" then if (. | contains("does not exist")) or (. | contains("not available")) then "⚠️  Expected: Debug APIs not available on public RPC" elif startswith("0x") then "✅ SUCCESS: Raw header returned" else . end else . end'

echo ""
echo "Test 7: debug_traceTransaction (ETH1 debug API) - Expected to fail with public RPC"
echo "(Note: This will fail if no recent transactions exist, which is expected)"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"debug_traceTransaction","params":["0x1234567890123456789012345678901234567890123456789012345678901234",{"tracer":"callTracer"}],"id":1}' | jq -r '.error // .result // "No transaction found (expected)"'

echo ""
echo "=============================================="
echo "Testing Archive Extension"
echo "=============================================="

echo ""
echo "Test 8: eth_getBlockByNumber with old block (should trigger archive)"
curl -s -X POST http://127.0.0.1:3360 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x1",false],"id":1}' | jq -r '.error // .result | if type == "object" then "✅ SUCCESS: Archive block retrieved" else . end'

echo ""
echo "=============================================="
echo "Add-on Verification Tests Complete"
echo "=============================================="
echo ""
echo "Expected Results with Public RPC:"
echo "  ✅ Standard APIs (eth_*) should work"
echo "  ✅ Trace add-on should work (trace_block, trace_transaction, trace_filter)"
echo "  ⚠️  Debug add-on expected to fail (requires dedicated RPC node with debug enabled)"
echo "  ✅ Archive extension should work for old blocks"
echo ""
echo "Notes:"
echo "  - Rate limiting from public RPC endpoints may cause some failures"
echo "  - Debug APIs (debug_*) require a dedicated/private RPC node"
echo "  - For production, use a dedicated node with debug APIs enabled"
echo ""
