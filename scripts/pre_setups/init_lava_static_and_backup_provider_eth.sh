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
sleep 1  # Give processes time to fully shut down before starting new ones

# Clean up any old generated provider configs in project root
echo "Cleaning up old provider configs..."
rm -f $PROJECT_ROOT/provider*_eth.yml 2>/dev/null || true

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

# Use absolute path for specs so screen sessions can find it
SPECS_DIR="$PROJECT_ROOT/specs/mainnet-1/specs/ethereum.json"
echo "Using static specs: $SPECS_DIR"

# Export RPC endpoint URLs as environment variables
# Set these before running the script:
#   export ETH_RPC_URL_1="https://mainnet.infura.io/v3/YOUR_INFURA_KEY"
#   export ETH_RPC_URL_2="https://purple-newest-dew.quiknode.pro/YOUR_QUICKNODE_KEY"
#   export ETH_RPC_URL_3="https://another-quiknode-endpoint.pro/YOUR_QUICKNODE_KEY"
#   
# Note: WebSocket endpoints are disabled in this configuration (HTTP only)

# Set defaults if not already exported (placeholders will fail to connect)
export ETH_RPC_URL_1="${ETH_RPC_URL_1:-https://mainnet.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export ETH_RPC_URL_2="${ETH_RPC_URL_2:-https://purple-newest-dew.quiknode.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export ETH_RPC_URL_3="${ETH_RPC_URL_3:-https://another-quiknode-endpoint.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"

# Validate that real URLs are set (not placeholders)
if [[ "$ETH_RPC_URL_1" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: ETH_RPC_URL_1 contains placeholder. Set real values with:"
    echo "  export ETH_RPC_URL_1='https://mainnet.infura.io/v3/YOUR_KEY'"
fi
if [[ "$ETH_RPC_URL_2" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: ETH_RPC_URL_2 contains placeholder. Set real values with:"
    echo "  export ETH_RPC_URL_2='https://purple-newest-dew.quiknode.pro/YOUR_KEY'"
fi
if [[ "$ETH_RPC_URL_3" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: ETH_RPC_URL_3 contains placeholder. Set real values with:"
    echo "  export ETH_RPC_URL_3='https://another-quiknode-endpoint.pro/YOUR_KEY'"
fi

# Generate provider configs on-the-fly in project root (where viper can find them)
# Viper searches in "." and "./config" by default, so we'll use project root
CONFIG_DIR="$PROJECT_ROOT"
echo "Generating configs in: $CONFIG_DIR"

echo "Generating provider configs with environment variables..."
echo "Config directory (absolute): $CONFIG_DIR"
echo "Environment variables:"
echo "  ETH_RPC_URL_1 (Infura HTTP):     ${ETH_RPC_URL_1:0:50}..."
echo "  ETH_RPC_URL_2 (QuickNode HTTP):  ${ETH_RPC_URL_2:0:50}..."
echo "  ETH_RPC_URL_3 (QuickNode HTTP):  ${ETH_RPC_URL_3:0:50}..."
echo ""
echo "Provider Endpoint Configuration (HTTP only - WebSocket disabled):"
echo "  Provider 1: Infura HTTP (archive)"
echo "  Provider 2: QuickNode HTTP (debug + archive)"
echo "  Provider 3: QuickNode HTTP endpoint 3 (debug + archive)"
echo ""
echo "Note: WebSocket endpoints are disabled to skip WS verification"
echo "      HTTP load is split: Provider 1 uses Infura, Providers 2&3 use QuickNode (separate endpoints)"

# Provider 1 config (based on eth_provider_with_archive_debug.yml)
# WebSocket URLs removed - HTTP only
cat > $CONFIG_DIR/provider1_eth.yml <<EOF
endpoints:
    - name: provider-archive
      api-interface: jsonrpc
      chain-id: ETH1
      network-address:
        address: "$PROVIDER1_LISTENER"
      node-urls:
        - url: $ETH_RPC_URL_1
        - url: $ETH_RPC_URL_1
          addons:
            - archive
EOF

# Provider 2 config (based on eth_provider_with_archive_debug1.yml)
# WebSocket URLs removed - HTTP only
cat > $CONFIG_DIR/provider2_eth.yml <<EOF
endpoints:
    - name: provider-debug-archive
      api-interface: jsonrpc
      chain-id: ETH1
      network-address:
        address: "$PROVIDER2_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $ETH_RPC_URL_2
        # Debug addon URLs
        - url: $ETH_RPC_URL_2
          addons:
            - debug
        # Archive addon URLs
        - url: $ETH_RPC_URL_2
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $ETH_RPC_URL_2
          addons:
            - debug
            - archive
EOF

# Provider 3 config (based on eth_provider_with_archive_debug2.yml)
# WebSocket URLs removed - HTTP only
cat > $CONFIG_DIR/provider3_eth.yml <<EOF
endpoints:
    - name: provider-debug-archive-2
      api-interface: jsonrpc
      chain-id: ETH1
      network-address:
        address: "$PROVIDER3_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $ETH_RPC_URL_3
        # Debug addon URLs
        - url: $ETH_RPC_URL_3
          addons:
            - debug
        # Archive addon URLs
        - url: $ETH_RPC_URL_3
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $ETH_RPC_URL_3
          addons:
            - debug
            - archive
EOF

echo "Provider configs generated successfully"

# Verify config files were created
echo ""
echo "Verifying generated config files..."
for i in 1 2 3; do
    CONFIG_FILE="$CONFIG_DIR/provider${i}_eth.yml"
    if [ -f "$CONFIG_FILE" ]; then
        FILE_SIZE=$(wc -c < "$CONFIG_FILE")
        echo "✓ Provider $i config exists: $CONFIG_FILE (size: $FILE_SIZE bytes)"
        # Show first few lines to verify content
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
# Provider 1 has 2 node-urls (HTTP only), Providers 2&3 have 4 node-urls each (HTTP only)
# With --parallel-connections 1, this opens limited connections during validation
# WebSocket endpoints are disabled - using --skip-websocket-verification flag to bypass WS requirement

# Start Provider 1 (archive, standalone mode, real ETH endpoint)
echo "[Test Setup] starting Provider 1 (archive, standalone mode)"
screen -d -m -S provider1 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider1_eth \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--skip-websocket-verification \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

echo "Waiting 3 seconds for Provider 1 to complete validation before starting Provider 2..."
sleep 3

# Start Provider 2 (archive, standalone mode, real ETH endpoint)
echo "[Test Setup] starting Provider 2 (archive, standalone mode)"
screen -d -m -S provider2 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider2_eth \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--skip-websocket-verification \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

echo "Waiting 3 seconds for Provider 2 to complete validation before starting Provider 3..."
sleep 3

# Start Provider 3 (archive, standalone mode, real ETH endpoint)
echo "[Test Setup] starting Provider 3 (archive, standalone mode)"
screen -d -m -S provider3 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider3_eth \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--skip-websocket-verification \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 2

# Verify providers started successfully
echo "Verifying provider screen sessions..."
sleep 1  # Give screens a moment to start
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
echo "[Test Setup] starting consumer (standalone mode)"
screen -d -m -S consumer bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcsmartrouter \
config/consumer_examples/lava_consumer_static_with_backup_eth.yml \
--geolocation 1 --log_level trace \
--allow-insecure-provider-dialing \
--use-static-spec $SPECS_DIR \
--metrics-listen-address ':7779' \
2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "Test Setup Complete (Fully Standalone Mode)"
echo "============================================"
echo "Consumer Cache:  127.0.0.1:20100 (metrics: 20200)"
echo "Provider Cache:  127.0.0.1:20101 (metrics: 20201)"
echo "Provider 1:      $PROVIDER1_LISTENER (Infura HTTP, archive)"
echo "Provider 2:      $PROVIDER2_LISTENER (QuickNode HTTP, debug+archive)"
echo "Provider 3:      $PROVIDER3_LISTENER (QuickNode HTTP endpoint 3, debug+archive)"
echo "Consumer:        rpcsmartrouter (fully standalone, cache-enabled)"
echo ""
echo "All components disconnected from Lava blockchain!"
echo "Using static specs: $SPECS_DIR"
echo "Logs: $LOGS_DIR"
echo ""
echo "Endpoint Strategy (HTTP only - WebSocket disabled):"
echo "  - Provider 1 uses Infura HTTP"
echo "  - Provider 2 uses QuickNode HTTP endpoint 2"
echo "  - Provider 3 uses QuickNode HTTP endpoint 3"
echo "  - Parallel connections: 1 per URL (avoids overwhelming endpoints)"
echo ""
echo "Cache Configuration:"
echo "  - Consumer uses cache at 127.0.0.1:20100"
echo "  - All providers share cache at 127.0.0.1:20101"
echo "============================================"
