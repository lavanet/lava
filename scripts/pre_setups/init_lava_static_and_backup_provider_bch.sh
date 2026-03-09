#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

# Use absolute paths for logs
LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
LOGS_DIR=$(cd "$LOGS_DIR" && pwd)
rm $LOGS_DIR/*.log

# Save project root for later use
PROJECT_ROOT=$(cd ${__dir}/../.. && pwd)

# Kill all lavap and lavad processes
killall lavap lavad 2>/dev/null || true
sleep 1

# Kill all screen sessions
killall screen 2>/dev/null || true
sleep 1
screen -wipe
sleep 1

# Clean up any old generated provider configs in project root
echo "Cleaning up old BCH provider configs..."
rm -f $PROJECT_ROOT/provider*_bch.yml 2>/dev/null || true

echo "[Test Setup] installing all binaries"
make install-all

# Start cache services (no blockchain needed for standalone)
echo "[Test Setup] starting consumer cache service"
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

echo "[Test Setup] starting provider cache service"
screen -d -m -S provider_cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20101 --metrics_address 0.0.0.0:20201 --log_level debug 2>&1 | tee $LOGS_DIR/PROVIDER_CACHE.log" && sleep 0.25

sleep 2

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

# Use absolute path for specs so screen sessions can find it.
# BCH imports BTC. Same as chains that import from ETH (e.g. BASE, BLAST):
# - Pass the specs *directory*: loader loads all JSON files and can resolve imports (BCH->BTC).
# - Alternatively comma-separated files (dependency first): btc.json,bch.json
# See protocol/statetracker/state_tracker.go (directory -> GetSpecFromLocalDir) and
# utils/keeper/spec.go (GetSpecsFromPath splits by comma, dependency order matters).
SPECS_DIR="$PROJECT_ROOT/specs/mainnet-1/specs"
echo "Using static specs (directory): $SPECS_DIR"

# Export BCH RPC endpoint URLs as environment variables
# Set these before running the script:
#   export BCH_RPC_URL_1="https://your-primary-bch-rpc.example.com"
#   export BCH_RPC_URL_2="https://your-secondary-bch-rpc.example.com"
#   export BCH_RPC_URL_3="https://your-tertiary-bch-rpc.example.com"
#
# BCH uses JSON-RPC over HTTP. We configure archive addon paths on each endpoint.

# Set defaults if not already exported (placeholders will fail to connect)
export BCH_RPC_URL_1="${BCH_RPC_URL_1:-https://your-primary-bch-rpc.example.com}"
export BCH_RPC_URL_2="${BCH_RPC_URL_2:-$BCH_RPC_URL_1}"
export BCH_RPC_URL_3="${BCH_RPC_URL_3:-$BCH_RPC_URL_1}"

# Validate that real URLs are set (not placeholders)
if [[ "$BCH_RPC_URL_1" == *"your-primary-bch-rpc.example.com"* ]]; then
    echo "Warning: BCH_RPC_URL_1 contains placeholder. Set real values with:"
    echo "  export BCH_RPC_URL_1='https://your-primary-bch-rpc.example.com'"
fi

# Generate provider configs on-the-fly in project root (where viper can find them)
CONFIG_DIR="$PROJECT_ROOT"
echo "Generating configs in: $CONFIG_DIR"

echo "Generating BCH provider configs with environment variables..."
echo "Config directory (absolute): $CONFIG_DIR"
echo "Environment variables:"
echo "  BCH_RPC_URL_1 (Provider 1):  ${BCH_RPC_URL_1:0:60}..."
echo "  BCH_RPC_URL_2 (Provider 2):  ${BCH_RPC_URL_2:0:60}..."
echo "  BCH_RPC_URL_3 (Provider 3):  ${BCH_RPC_URL_3:0:60}..."
echo ""

# Provider 1 config (archive)
cat > $CONFIG_DIR/provider1_bch.yml <<EOF
endpoints:
    - name: provider-archive
      api-interface: jsonrpc
      chain-id: BCH
      network-address:
        address: "$PROVIDER1_LISTENER"
      node-urls:
        - url: $BCH_RPC_URL_1
        - url: $BCH_RPC_URL_1
          addons:
            - archive
EOF

# Provider 2 config (archive)
cat > $CONFIG_DIR/provider2_bch.yml <<EOF
endpoints:
    - name: provider-archive-2
      api-interface: jsonrpc
      chain-id: BCH
      network-address:
        address: "$PROVIDER2_LISTENER"
      node-urls:
        - url: $BCH_RPC_URL_2
        - url: $BCH_RPC_URL_2
          addons:
            - archive
EOF

# Provider 3 config (archive)
cat > $CONFIG_DIR/provider3_bch.yml <<EOF
endpoints:
    - name: provider-archive-3
      api-interface: jsonrpc
      chain-id: BCH
      network-address:
        address: "$PROVIDER3_LISTENER"
      node-urls:
        - url: $BCH_RPC_URL_3
        - url: $BCH_RPC_URL_3
          addons:
            - archive
EOF

echo "Provider configs generated successfully"

# Verify config files were created
echo ""
echo "Verifying generated config files..."
for i in 1 2 3; do
    CONFIG_FILE="$CONFIG_DIR/provider${i}_bch.yml"
    if [ -f "$CONFIG_FILE" ]; then
        FILE_SIZE=$(wc -c < "$CONFIG_FILE")
        echo "✓ Provider $i config exists: $CONFIG_FILE (size: $FILE_SIZE bytes)"
        echo "  First 3 lines:"
        head -n 3 "$CONFIG_FILE" | sed 's/^/    /'
    else
        echo "✗ ERROR: Provider $i config NOT found: $CONFIG_FILE"
        echo "  Directory contents:"
        ls -la "$CONFIG_DIR"
        exit 1
    fi
done
echo ""

# Note: Using --parallel-connections 1 to limit connections per URL

# Start Provider 1 (archive, standalone mode, real BCH endpoint)
echo "[Test Setup] starting Provider 1 (archive, standalone mode)"
screen -d -m -S provider1 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider1_bch \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

echo "Waiting 3 seconds for Provider 1 to complete validation before starting Provider 2..."
sleep 3

# Start Provider 2 (archive, standalone mode, real BCH endpoint)
echo "[Test Setup] starting Provider 2 (archive, standalone mode)"
screen -d -m -S provider2 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider2_bch \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

echo "Waiting 3 seconds for Provider 2 to complete validation before starting Provider 3..."
sleep 3

# Start Provider 3 (archive, standalone mode, real BCH endpoint)
echo "[Test Setup] starting Provider 3 (archive, standalone mode)"
screen -d -m -S provider3 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider3_bch \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 2

# Verify providers started successfully
echo "Verifying provider screen sessions..."
sleep 1
for i in 1 2 3; do
    provider_name="provider$i"
    log_file="PROVIDER${i}.log"
    if screen -list | grep -q "$provider_name"; then
        echo "✓ $provider_name screen is running"
    else
        echo "✗ ERROR: $provider_name screen failed to start!"
        echo "  Check $LOGS_DIR/$log_file for errors"
    fi
done
echo ""

# Start consumer (rpcsmartrouter - standalone mode, works with static providers)
echo "[Test Setup] starting consumer (rpcsmartrouter with cache, standalone mode)"
screen -d -m -S consumer bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_static_with_backup_bch.yml \
--geolocation 1 --log_level trace \
--cache-be "127.0.0.1:20100" \
--allow-insecure-provider-dialing \
--use-static-spec $SPECS_DIR \
--metrics-listen-address ':7779' \
--enable-provider-optimizer-auto-adjustment-of-tiers 2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "Test Setup Complete (BCH Standalone Mode)"
echo "============================================"
echo "Consumer Cache:  127.0.0.1:20100 (metrics: 20200)"
echo "Provider Cache:  127.0.0.1:20101 (metrics: 20201)"
echo "Provider 1:      $PROVIDER1_LISTENER (archive)"
echo "Provider 2:      $PROVIDER2_LISTENER (archive)"
echo "Provider 3:      $PROVIDER3_LISTENER (archive)"
echo "Consumer:        rpcsmartrouter (standalone, cache-enabled)"
echo ""
echo "Using static specs: $SPECS_DIR"
echo "Logs: $LOGS_DIR"
echo "============================================"
