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
rm -f $PROJECT_ROOT/provider*_base.yml 2>/dev/null || true

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
# Base imports ETH1, so we need to load both ethereum.json and base.json
ETH_SPEC="$PROJECT_ROOT/specs/mainnet-1/specs/ethereum.json"
BASE_SPEC="$PROJECT_ROOT/specs/mainnet-1/specs/base.json"
SPECS_DIR="$ETH_SPEC,$BASE_SPEC"
echo "Using static specs: $SPECS_DIR"

# Export RPC endpoint URLs as environment variables
# Set these before running the script:
#   export BASE_RPC_URL_1="https://base-mainnet.infura.io/v3/YOUR_INFURA_KEY"
#   export BASE_RPC_URL_2="https://base-endpoint.quiknode.pro/YOUR_QUICKNODE_KEY"
#   export BASE_RPC_URL_3="https://another-base-endpoint.pro/YOUR_QUICKNODE_KEY"
#   
# Optional WebSocket support (for subscription methods like eth_subscribe):
#   export BASE_RPC_WS_1="wss://base-mainnet.infura.io/ws/v3/YOUR_INFURA_KEY"
#   (If not set, providers will use HTTP only)
#   (BASE_RPC_WS_2/3 not used - Providers 2&3 use Infura WS instead when enabled)
#   
# Note: QuickNode free tier has a 2 WebSocket connection limit
# Solution: When WebSocket is enabled, all providers use Infura WebSocket ($BASE_RPC_WS_1)
# Only HTTP endpoints use QuickNode to spread the load

# Set defaults if not already exported (placeholders will fail to connect)
export BASE_RPC_URL_1="${BASE_RPC_URL_1:-https://base-mainnet.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export BASE_RPC_WS_1="${BASE_RPC_WS_1:-}"
export BASE_RPC_URL_2="${BASE_RPC_URL_2:-https://base-endpoint.quiknode.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export BASE_RPC_URL_3="${BASE_RPC_URL_3:-https://another-base-endpoint.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
# BASE_RPC_WS_2/3 not used in current config (Providers 2&3 are HTTP only)

# Validate that real URLs are set (not placeholders)
if [[ "$BASE_RPC_URL_1" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: BASE_RPC_URL_1 contains placeholder. Set real values with:"
    echo "  export BASE_RPC_URL_1='https://base-mainnet.infura.io/v3/YOUR_KEY'"
fi
if [[ "$BASE_RPC_URL_2" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: BASE_RPC_URL_2 contains placeholder. Set real values with:"
    echo "  export BASE_RPC_URL_2='https://base-endpoint.quiknode.pro/YOUR_KEY'"
fi
if [[ "$BASE_RPC_URL_3" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: BASE_RPC_URL_3 contains placeholder. Set real values with:"
    echo "  export BASE_RPC_URL_3='https://another-base-endpoint.pro/YOUR_KEY'"
fi

# WebSocket is optional - check if it's provided and valid
USE_WEBSOCKET=false
if [[ -n "$BASE_RPC_WS_1" ]]; then
    if [[ "$BASE_RPC_WS_1" =~ ^wss?:// ]]; then
        USE_WEBSOCKET=true
        echo "WebSocket enabled: $BASE_RPC_WS_1"
    else
        echo "Warning: BASE_RPC_WS_1 is set but not a valid ws/wss URL. Ignoring WebSocket."
        echo "  Got: $BASE_RPC_WS_1"
        echo "  Set it with: export BASE_RPC_WS_1='wss://base-mainnet.infura.io/ws/v3/YOUR_KEY'"
    fi
else
    echo "WebSocket disabled (BASE_RPC_WS_1 not set). Only HTTP endpoints will be used."
    echo "  Note: Subscription methods (eth_subscribe, etc.) will not be available."
    echo "  To enable WebSocket: export BASE_RPC_WS_1='wss://base-mainnet.infura.io/ws/v3/YOUR_KEY'"
fi

# Generate provider configs on-the-fly in project root (where viper can find them)
# Viper searches in "." and "./config" by default, so we'll use project root
CONFIG_DIR="$PROJECT_ROOT"
echo "Generating configs in: $CONFIG_DIR"

echo "Generating provider configs with environment variables..."
echo "Config directory (absolute): $CONFIG_DIR"
echo "Environment variables:"
echo "  BASE_RPC_URL_1 (Infura HTTP):     ${BASE_RPC_URL_1:0:50}..."
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    echo "  BASE_RPC_WS_1 (Infura WS):        ${BASE_RPC_WS_1:0:50}..."
fi
echo "  BASE_RPC_URL_2 (QuickNode HTTP):  ${BASE_RPC_URL_2:0:50}..."
echo "  BASE_RPC_URL_3 (QuickNode HTTP):  ${BASE_RPC_URL_3:0:50}..."
echo ""
echo "Provider Endpoint Configuration:"
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    echo "  Provider 1: Infura HTTP + Infura WS (archive)"
    echo "  Provider 2: QuickNode HTTP + Infura WS (debug + archive)"
    echo "  Provider 3: QuickNode HTTP (endpoint 3) + Infura WS (debug + archive)"
    echo ""
    echo "Note: All providers use Infura WebSocket to avoid QuickNode's 2 WS connection limit"
    echo "      HTTP load is split: Provider 1 uses Infura, Providers 2&3 use QuickNode (separate endpoints)"
else
    echo "  Provider 1: Infura HTTP only (archive)"
    echo "  Provider 2: QuickNode HTTP only (debug + archive)"
    echo "  Provider 3: QuickNode HTTP only (endpoint 3, debug + archive)"
    echo ""
    echo "Note: WebSocket disabled - subscription methods will not be available"
    echo "      HTTP load is split: Provider 1 uses Infura, Providers 2&3 use QuickNode (separate endpoints)"
fi

# Provider 1 config (based on base_provider_with_archive_debug.yml)
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    cat > $CONFIG_DIR/provider1_base.yml <<EOF
endpoints:
    - name: provider-archive
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER1_LISTENER"
      node-urls:
        - url: $BASE_RPC_URL_1
        - url: $BASE_RPC_WS_1
        - url: $BASE_RPC_URL_1
          addons:
            - archive
        - url: $BASE_RPC_WS_1
          addons:
            - archive
EOF
else
    cat > $CONFIG_DIR/provider1_base.yml <<EOF
endpoints:
    - name: provider-archive
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER1_LISTENER"
      node-urls:
        - url: $BASE_RPC_URL_1
        - url: $BASE_RPC_URL_1
          addons:
            - archive
EOF
fi

# Provider 2 config (based on base_provider_with_archive_debug1.yml)
# Uses QuickNode HTTP + Infura WebSocket (to avoid QuickNode WS limit) if WebSocket is enabled
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    cat > $CONFIG_DIR/provider2_base.yml <<EOF
endpoints:
    - name: provider-debug-archive
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER2_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $BASE_RPC_URL_2
        - url: $BASE_RPC_WS_1
        # Debug addon URLs
        - url: $BASE_RPC_URL_2
          addons:
            - debug
        - url: $BASE_RPC_WS_1
          addons:
            - debug
        # Archive addon URLs
        - url: $BASE_RPC_URL_2
          addons:
            - archive
        - url: $BASE_RPC_WS_1
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $BASE_RPC_URL_2
          addons:
            - debug
            - archive
        - url: $BASE_RPC_WS_1
          addons:
            - debug
            - archive
EOF
else
    cat > $CONFIG_DIR/provider2_base.yml <<EOF
endpoints:
    - name: provider-debug-archive
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER2_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $BASE_RPC_URL_2
        # Debug addon URLs
        - url: $BASE_RPC_URL_2
          addons:
            - debug
        # Archive addon URLs
        - url: $BASE_RPC_URL_2
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $BASE_RPC_URL_2
          addons:
            - debug
            - archive
EOF
fi

# Provider 3 config (based on base_provider_with_archive_debug2.yml)
# Uses QuickNode HTTP (endpoint 3) + Infura WebSocket (to avoid QuickNode WS limit) if WebSocket is enabled
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    cat > $CONFIG_DIR/provider3_base.yml <<EOF
endpoints:
    - name: provider-debug-archive-2
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER3_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $BASE_RPC_URL_3
        - url: $BASE_RPC_WS_1
        # Debug addon URLs
        - url: $BASE_RPC_URL_3
          addons:
            - debug
        - url: $BASE_RPC_WS_1
          addons:
            - debug
        # Archive addon URLs
        - url: $BASE_RPC_URL_3
          addons:
            - archive
        - url: $BASE_RPC_WS_1
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $BASE_RPC_URL_3
          addons:
            - debug
            - archive
        - url: $BASE_RPC_WS_1
          addons:
            - debug
            - archive
EOF
else
    cat > $CONFIG_DIR/provider3_base.yml <<EOF
endpoints:
    - name: provider-debug-archive-2
      api-interface: jsonrpc
      chain-id: BASE
      network-address:
        address: "$PROVIDER3_LISTENER"
      node-urls:
        # Base URLs (no addons) - for regular requests
        - url: $BASE_RPC_URL_3
        # Debug addon URLs
        - url: $BASE_RPC_URL_3
          addons:
            - debug
        # Archive addon URLs
        - url: $BASE_RPC_URL_3
          addons:
            - archive
        # Combined debug+archive URLs
        - url: $BASE_RPC_URL_3
          addons:
            - debug
            - archive
EOF
fi

echo "Provider configs generated successfully"

# Verify config files were created
echo ""
echo "Verifying generated config files..."
for i in 1 2 3; do
    CONFIG_FILE="$CONFIG_DIR/provider${i}_base.yml"
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
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    # Each provider has 8 node-urls (4 HTTP + 4 WS), with --parallel-connections 1
    # this opens max 8 connections during validation (4 HTTP + 4 WS)
    # WebSocket strategy: All use Infura WS to avoid QuickNode's 2 WS connection limit
    echo "Connection strategy: HTTP + WebSocket (max 8 connections per provider during validation)"
else
    # Each provider has 4 node-urls (HTTP only), with --parallel-connections 1
    # this opens max 4 connections during validation
    echo "Connection strategy: HTTP only (max 4 connections per provider during validation)"
fi

# Start Provider 1 (archive, standalone mode, real BASE endpoint)
echo "[Test Setup] starting Provider 1 (archive, standalone mode)"
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    screen -d -m -S provider1 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider1_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
else
    screen -d -m -S provider1 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider1_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--skip-websocket-verification \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7777' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
fi

echo "Waiting 3 seconds for Provider 1 to complete validation before starting Provider 2..."
sleep 3

# Start Provider 2 (archive, standalone mode, real BASE endpoint)
echo "[Test Setup] starting Provider 2 (archive, standalone mode)"
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    screen -d -m -S provider2 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider2_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25
else
    screen -d -m -S provider2 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider2_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--skip-websocket-verification \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25
fi

echo "Waiting 3 seconds for Provider 2 to complete validation before starting Provider 3..."
sleep 3

# Start Provider 3 (archive, standalone mode, real BASE endpoint)
echo "[Test Setup] starting Provider 3 (archive, standalone mode)"
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    screen -d -m -S provider3 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider3_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25
else
    screen -d -m -S provider3 bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcprovider \
provider3_base \
--static-providers \
--use-static-spec $SPECS_DIR \
--parallel-connections 1 \
--skip-websocket-verification \
--cache-be \"127.0.0.1:20101\" \
--geolocation 1 --log_level debug --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25
fi

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
config/consumer_examples/lava_consumer_static_with_backup_base.yml \
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
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    echo "Provider 1:      $PROVIDER1_LISTENER (Infura HTTP + Infura WS, archive)"
    echo "Provider 2:      $PROVIDER2_LISTENER (QuickNode HTTP + Infura WS, debug+archive)"
    echo "Provider 3:      $PROVIDER3_LISTENER (QuickNode HTTP endpoint 3 + Infura WS, debug+archive)"
else
    echo "Provider 1:      $PROVIDER1_LISTENER (Infura HTTP only, archive)"
    echo "Provider 2:      $PROVIDER2_LISTENER (QuickNode HTTP only, debug+archive)"
    echo "Provider 3:      $PROVIDER3_LISTENER (QuickNode HTTP endpoint 3 only, debug+archive)"
fi
echo "Consumer:        rpcsmartrouter (fully standalone, cache-enabled)"
echo ""
echo "All components disconnected from Lava blockchain!"
echo "Using static specs: $SPECS_DIR"
echo "Logs: $LOGS_DIR"
echo ""
echo "Endpoint Strategy:"
if [[ "$USE_WEBSOCKET" == "true" ]]; then
    echo "  - WebSocket: Enabled (Infura - avoids QuickNode's 2 WS limit)"
    echo "  - HTTP: Provider 1 uses Infura, Provider 2 uses QuickNode endpoint 2, Provider 3 uses endpoint 3"
else
    echo "  - WebSocket: Disabled (HTTP only)"
    echo "  - HTTP: Provider 1 uses Infura, Provider 2 uses QuickNode endpoint 2, Provider 3 uses endpoint 3"
    echo "  - Note: Subscription methods (eth_subscribe, etc.) will not be available"
fi
echo "  - Parallel connections: 1 per URL (avoids overwhelming endpoints)"
echo ""
echo "Cache Configuration:"
echo "  - Consumer uses cache at 127.0.0.1:20100"
echo "  - All providers share cache at 127.0.0.1:20101"
echo "============================================"
