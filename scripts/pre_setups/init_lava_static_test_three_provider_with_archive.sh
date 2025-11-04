#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe
sleep 2  # Give processes time to fully shut down before starting new ones

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

# Start Provider 1 (non-archive, test mode, standalone)
echo "[Test Setup] starting Provider 1 (non-archive, standalone mode)"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider1_noarchive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
--static-providers \
--use-static-spec specs/mainnet-1/specs/ \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

# Start Provider 2 (non-archive, test mode, standalone)
echo "[Test Setup] starting Provider 2 (non-archive, standalone mode)"
screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/provider2_noarchive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_noarchive.json \
--static-providers \
--use-static-spec specs/mainnet-1/specs/ \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

# Start Provider 3 (archive, test mode, standalone)
echo "[Test Setup] starting Provider 3 (archive, standalone mode)"
screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
config/provider_examples/lava_example_archive.yml \
--test_mode --test_responses ./scripts/test_data/test_responses_jsonrpc_archive.json \
--static-providers \
--use-static-spec specs/mainnet-1/specs/ \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 2

# Start consumer (rpcsmartrouter - standalone mode, works with static providers)
echo "[Test Setup] starting consumer (rpcsmartrouter with cache, standalone mode)"
screen -d -m -S consumer bash -c "cd /Users/anna/go/lava && source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_static_peers.yml \
--geolocation 1 --log_level trace --debug-relays \
--cache-be 127.0.0.1:20100 \
--allow-insecure-provider-dialing \
--use-static-spec specs/mainnet-1/specs/ \
--metrics-listen-address ':7779' 2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "Test Setup Complete (Fully Standalone Mode)"
echo "============================================"
echo "Cache:      127.0.0.1:20100"
echo "Provider 1: $PROVIDER1_LISTENER (non-archive, fully standalone)"
echo "Provider 2: $PROVIDER2_LISTENER (non-archive, fully standalone)"
echo "Provider 3: $PROVIDER3_LISTENER (archive, fully standalone)"
echo "Consumer:   rpcsmartrouter (fully standalone, cache-enabled)"
echo ""
echo "All components disconnected from Lava blockchain!"
echo "Using static specs: specs/mainnet-1/specs/"
echo "Logs: $LOGS_DIR"
echo "============================================"
