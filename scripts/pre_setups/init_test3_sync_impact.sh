#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

# Kill all lavap and lavad processes
killall lavap lavad 2>/dev/null || true
sleep 1

# Kill all screen sessions
killall screen 2>/dev/null || true
sleep 1
screen -wipe
sleep 1

echo "============================================"
echo "TEST 3: SYNC IMPACT TEST"
echo "============================================"
echo "Testing sync parameter impact on provider selection"
echo "Provider configs:"
echo "  - Provider 1 (2220): SYNCED (block 1000, 0 blocks behind)"
echo "  - Provider 2 (2221): LAGGING (block 999, 1 block behind)"
echo "  - Provider 3 (2222): VERY_LAGGING (block 998, 2 blocks behind)"
echo ""
echo "Weights: Sync=0.5, Availability=0.5, Latency=0, Stake=0"
echo "============================================"
echo ""

echo "[Test 3] Installing binaries..."
make install-all 

# Start cache service
echo "[Test 3] Starting cache service..."
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

sleep 2

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

SPECS_DIR="./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/testnet-2/specs/lava.json"

# PHASE 1: Start ONLY Provider 1 first to establish latestSync = 1000
echo "[Test 3] PHASE 1: Starting Provider 1 ONLY (tendermintrpc interface only for fast startup)..."
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider1_test3_tendermintrpc_only.yml \
--test_mode --test_responses ./wrs_test_configs/test3_synced.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

echo "[Test 3] Waiting for Provider 1 to be ready (tendermintrpc only, ~10 seconds)..."
sleep 12

echo "[Test 3] Provider 1 should now be fully ready (tendermintrpc endpoint initialized)"
echo ""

# Start consumer FIRST (before sending init requests) so it can track latestSync
echo "[Test 3] Starting consumer (to track latestSync)..."
screen -d -m -S consumer bash -c "source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_static_peers.yml \
--geolocation 1 --log_level trace --debug-relays --enable-selection-stats \
--cache-be 127.0.0.1:20100 \
--allow-insecure-provider-dialing \
--use-static-spec $SPECS_DIR \
--provider-optimizer-sync-weight 0.5 \
--provider-optimizer-availability-weight 0.5 \
--provider-optimizer-latency-weight 0 \
--provider-optimizer-stake-weight 0 \
--metrics-listen-address :7779 \
--optimizer-qos-listen 2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "[Test 3] Waiting for consumer to be ready (this takes ~30-40 seconds)..."
sleep 35

echo "[Test 3] Sending initialization requests to Provider 1 (establish latestSync = 1000)..."
SUCCESS_COUNT=0
for i in {1..20}; do
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "lava-force-cache-refresh: true" \
    --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
    http://127.0.0.1:3360/1/lava/tendermintrpc/LAV1 2>/dev/null)
  
  if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    echo -n "✓"
  else
    echo -n "✗"
  fi
done
echo ""
echo "  Initialization complete: $SUCCESS_COUNT/20 requests succeeded"

if [ $SUCCESS_COUNT -lt 10 ]; then
  echo "✗ WARNING: Less than 50% of initialization requests succeeded"
  echo "  latestSync might not be properly established at block 1000"
fi
echo "  latestSync should now be 1000"
echo ""

# PHASE 2: Now start Provider 2 and Provider 3 with lagged blocks
echo "[Test 3] PHASE 2: Starting Provider 2 & 3 (lagging providers)..."

# Start Provider 2 - LAGGING (block 999, 1 block behind)
echo "[Test 3] Starting Provider 2 (LAGGING - block 999, 1 behind)..."
screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider2_noarchive.yml \
--test_mode --test_responses ./wrs_test_configs/test3_lagging.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

# Start Provider 3 - VERY_LAGGING (block 998, 2 blocks behind)
echo "[Test 3] Starting Provider 3 (VERY_LAGGING - block 998, 2 behind)..."
screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/lava_example_archive.yml \
--test_mode --test_responses ./wrs_test_configs/test3_very_lagging.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

echo "[Test 3] Waiting for Provider 2 & 3 to be ready..."
sleep 3

echo "--- Test 3 setup complete ---"
screen -ls

echo ""
echo "============================================"
echo "TEST 3 Setup Complete (Sync Impact Test)"
echo "============================================"
echo "PHASE 1: Provider 1 started first with 20 initialization requests"
echo "         latestSync established at block 1000"
echo ""
echo "PHASE 2: Provider 2 & 3 started with lagged blocks"
echo ""
echo "Cache:      127.0.0.1:20100"
echo "Provider 1: $PROVIDER1_LISTENER (SYNCED - block 1000, 0 behind)"
echo "Provider 2: $PROVIDER2_LISTENER (LAGGING - block 999, 1 behind)"
echo "Provider 3: $PROVIDER3_LISTENER (VERY_LAGGING - block 998, 2 behind)"
echo "Consumer:   rpcsmartrouter"
echo ""
echo "Weights:"
echo "  - Sync:         0.5 (50%)"
echo "  - Availability: 0.5 (50%)"
echo "  - Latency:      0.0 (0%)"
echo "  - Stake:        0.0 (0%)"
echo ""
echo "Expected Results:"
echo "  Provider 1 (synced):       ~41.7% (score 1.0,  normalized_sync=1.0)"
echo "  Provider 2 (1 block lag):  ~33.3% (score 0.80, normalized_sync=0.6)"
echo "  Provider 3 (2 blocks lag): ~25.0% (score 0.60, normalized_sync=0.2)"
echo ""
echo "Optimizer QoS Metrics: http://localhost:7779/provider_optimizer_metrics"
echo "Logs: $LOGS_DIR"
echo "============================================"
echo ""
echo "Waiting 5 seconds for all services to stabilize..."
sleep 5
echo ""
echo "Sending warm-up requests (30)..."
for i in {1..30}; do
  curl -s -X POST -H "Content-Type: application/json" -H "lava-force-cache-refresh: true" \
    --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
    http://127.0.0.1:3360/1/lava/tendermintrpc/LAV1 > /dev/null 2>&1
  echo -n "."
done
echo " Done"
echo ""
echo "Sending test requests (500)..."
for i in {1..500}; do
  curl -s -X POST -H "Content-Type: application/json" -H "lava-force-cache-refresh: true" \
    --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
    http://127.0.0.1:3360/1/lava/tendermintrpc/LAV1 > /dev/null 2>&1
  if [ $((i % 50)) -eq 0 ]; then
    echo "  Progress: $i/500..."
  fi
done
echo "Test requests complete!"
echo ""
echo "============================================"
echo "Analyzing Results..."
echo "============================================"
echo ""
echo "Provider Selection Distribution:"
grep "Provider selection completed" $LOGS_DIR/CONSUMER.log | \
  grep -o 'selected_provider=[^ ]*' | \
  cut -d= -f2 | sort | uniq -c | \
  awk '{printf "  %s: %d selections (%.1f%%)\n", $2, $1, ($1/550)*100}'
echo ""
echo "Check detailed logs:"
echo "  Score calculations: grep 'Provider score calculation breakdown' $LOGS_DIR/CONSUMER.log"
echo "  Selections:        grep 'Provider selection completed' $LOGS_DIR/CONSUMER.log"
echo ""
echo "Check metrics endpoint:"
echo "  curl http://localhost:7779/provider_optimizer_metrics | python3 -m json.tool"
echo ""
