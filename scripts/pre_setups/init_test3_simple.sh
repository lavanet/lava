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
sleep 1  # Give processes time to fully shut down before starting new ones

echo "[Test Setup] installing all binaries"
make install-all 

# Start cache service (no blockchain needed for standalone)
echo "[Test Setup] starting cache service"
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

sleep 2

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

SPECS_DIR="./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/testnet-2/specs/lava.json"
echo "Using static specs: $SPECS_DIR"
# Start Provider 1 (SYNCED: gap=0, test mode, standalone)
echo "[Test 3] starting Provider 1 (SYNCED - block 1000, gap=0)"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider1_test3_tendermintrpc_only.yml \
--test_mode --test_responses ./wrs_test_configs/test3_synced.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level info --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

# Start Provider 2 (LAGGING: gap=1, test mode, standalone)
echo "[Test 3] starting Provider 2 (LAGGING - block 999, gap=1)"
screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider2_test3_tendermintrpc_only.yml \
--test_mode --test_responses ./wrs_test_configs/test3_lagging.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level info --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

# Start Provider 3 (VERY_LAGGING: gap=2, test mode, standalone)
echo "[Test 3] starting Provider 3 (VERY_LAGGING - block 998, gap=2)"
screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider3_test3_tendermintrpc_only.yml \
--test_mode --test_responses ./wrs_test_configs/test3_very_lagging.json \
--static-providers \
--use-static-spec $SPECS_DIR \
--geolocation 1 --log_level info --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 2

# Start consumer (rpcsmartrouter - standalone mode, Test 3 config)
echo "[Test 3] starting consumer (tendermintrpc only)"
echo "[Test 3] Testing SYNC parameter impact with weights: Sync=0.5, Availability=0.5"
screen -d -m -S consumer bash -c "cd /Users/anna/go/lava && source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_test3_tendermintrpc_only.yml \
--geolocation 1 --log_level info --debug-relays \
--cache-be 127.0.0.1:20100 \
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
echo "Cache:      127.0.0.1:20100"
echo "Provider 1: $PROVIDER1_LISTENER (SYNCED - block 1000, gap=0)"
echo "Provider 2: $PROVIDER2_LISTENER (LAGGING - block 999, gap=1)"
echo "Provider 3: $PROVIDER3_LISTENER (VERY_LAGGING - block 998, gap=2)"
echo "Consumer:   rpcsmartrouter (tendermintrpc only)"
echo ""
echo "Test Mode: Deterministic baseline"
echo "  - Send FIRST request with header: lava-select-provider: provider-2220-tendermintrpc"
echo "  - This pins the baseline to the synced provider (gap=0)"
echo ""
echo "Weights: Sync=0.5 (50%), Availability=0.5 (50%), Latency=0, Stake=0"
echo ""
echo "Expected Sync Lag:"
echo "  Provider 1 (gap=0): ~0s → normalized_sync ≈ 1.0"
echo "  Provider 2 (gap=1): ~12s → normalized_sync ≈ 0.99"  
echo "  Provider 3 (gap=2): ~24s → normalized_sync ≈ 0.98"
echo ""
echo "📊 Optimizer Metrics: http://localhost:7779/provider_optimizer_metrics"
echo "Logs: $LOGS_DIR"
echo "============================================"
