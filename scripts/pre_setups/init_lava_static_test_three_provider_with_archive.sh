#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm -f $LOGS_DIR/*.log

FIRST_PROVIDER="provider-2220-tendermintrpc"

# Kill all lavap and lavad processes
killall lavap lavad 2>/dev/null || true
sleep 1

# Kill all screen sessions
killall screen 2>/dev/null || true
sleep 1
screen -wipe
sleep 1  # Give processes time to fully shut down before starting new ones

echo "[Test Setup] installing all binaries"
make install-all 

IGNORE_CACHE="${IGNORE_CACHE:-true}"
CONSUMER_CACHE_ARGS=""
if [ "$IGNORE_CACHE" = false ]; then
  # Start cache service (no blockchain needed for standalone)
  echo "[Test Setup] starting cache service"
  screen -d -m -S cache bash -c "cd /Users/anna/go/lava && source ~/.bashrc; lavap cache \
  127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25
  sleep 2
  CONSUMER_CACHE_ARGS="--cache-be 127.0.0.1:20100"
else
  echo "[Test Setup] cache disabled (IGNORE_CACHE=true)"
fi

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

# For Test 3 (sync impact), load the full spec set (LAV1 imports other specs).
SPECS_DIR="./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/testnet-2/specs/lava.json"
echo "Using static specs: $SPECS_DIR"
# Start Provider 1 (non-archive, test mode, standalone)
echo "[Test Setup] starting Provider 1 (non-archive, standalone mode)"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider1_noarchive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level trace --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

# Start Provider 2 (non-archive, test mode, standalone)
echo "[Test Setup] starting Provider 2 (non-archive, standalone mode)"
screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider2_noarchive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level trace --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

# Start Provider 3 (VERY_LAGGING: gap=2, archive config, test mode, standalone)
echo "[Test 3] starting Provider 3 (VERY_LAGGING - head_block=1000, gap=2, archive endpoints)"
screen -d -m -S provider3 bash -c "cd /Users/anna/go/lava && source ~/.bashrc; lavap rpcprovider \
config/provider_examples/lava_example_archive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_archive.json \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level trace --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 2

# Start consumer (rpcsmartrouter - standalone mode, Test 3 config)
echo "[Test 3] starting consumer (tendermintrpc only)"
echo "[Test 3] Testing SYNC parameter impact with weights: Sync=0.5, Availability=0.5"
screen -d -m -S consumer bash -c "cd /Users/anna/go/lava && source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_test3_tendermintrpc_only.yml \
--geolocation 1 --log_level trace --debug-relays --enable-selection-stats \
$CONSUMER_CACHE_ARGS \
--allow-insecure-provider-dialing \
--use-static-spec $SPECS_DIR \
--metrics-listen-address ':7779' \
--optimizer-qos-listen \
--provider-optimizer-availability-weight 0.5 \
--provider-optimizer-latency-weight 0.0 \
--provider-optimizer-sync-weight 0.5 \
--provider-optimizer-stake-weight 0.0 \
--provider-optimizer-min-selection-chance 0.01 \
2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "TEST 3: Sync Impact Test (First relay pinned via lava-select-provider)"
echo "============================================"
if [ "$IGNORE_CACHE" = false ]; then
  echo "Cache:      127.0.0.1:20100"
else
  echo "Cache:      disabled"
fi
echo "Provider 1: $PROVIDER1_LISTENER (SYNCED - head_block=1000, gap=0)"
echo "Provider 2: $PROVIDER2_LISTENER (LAGGING - head_block=1000, gap=1)"
echo "Provider 3: $PROVIDER3_LISTENER (VERY_LAGGING - head_block=1000, gap=2, archive endpoints)"
echo "Consumer:   rpcsmartrouter (tendermintrpc only)"
echo ""
echo "All components disconnected from Lava blockchain!"
echo "Using static spec: $SPECS_DIR"
echo "Logs: $LOGS_DIR"
echo "============================================"
echo ""
echo "Quick check (tendermintrpc health) via consumer:"
echo "  # Pin first relay to synced provider to deterministically establish baseline:"
echo "  curl -s -i -X POST http://127.0.0.1:3361 -H 'Content-Type: application/json' -H \"lava-select-provider: $FIRST_PROVIDER\" -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"health\",\"params\":[]}' | egrep -i 'lava-provider-address|lava-selection-stats|provider-latest-block'"
echo ""
echo "Load test (prints provider distribution):"
echo "  ./scripts/e2e/run_optimizer_wrs_health_load.sh 50 http://127.0.0.1:3361"
