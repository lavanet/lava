#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

# Use absolute paths for logs
LOGS_DIR=${__dir}/../../debugging/logs
mkdir -p $LOGS_DIR
LOGS_DIR=$(cd "$LOGS_DIR" && pwd)
rm $LOGS_DIR/*.log 2>/dev/null || true

# Save project root for later use
PROJECT_ROOT=$(cd ${__dir}/../.. && pwd)
CONFIG_FILE="$PROJECT_ROOT/smartrouter_eth.yml"

# Only remove config when explicitly regenerating (keeps manual edits for e.g. timeout testing)
if [[ "$REGENERATE_CONFIG" == "1" ]]; then
    echo "REGENERATE_CONFIG=1: removing existing smart router config..."
    rm -f "$CONFIG_FILE" 2>/dev/null || true
fi

# Kill all lavap and lavad processes
killall lavap lavad 2>/dev/null || true
sleep 1

# Kill all screen sessions
killall screen 2>/dev/null || true
sleep 1
screen -wipe
sleep 1  # Give processes time to fully shut down before starting new ones

echo "============================================"
echo "Smart Router Direct RPC Test Setup"
echo "============================================"
echo "Testing: Phases 1-5 (JSON-RPC + WebSocket)"
echo "Mode: DIRECT RPC (no providers!)"
echo "============================================"
echo ""

echo "[Test Setup] installing all binaries"
make install-all

# Start cache services (required for cache testing)
echo "[Test Setup] starting smart router cache service"
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

sleep 2

# Verify cache service started
echo "Verifying cache service..."
if screen -list | grep -q "cache"; then
    echo "  Cache screen session: RUNNING"
    # Check if cache is listening (give it a moment)
    sleep 1
    if nc -z 127.0.0.1 20100 2>/dev/null; then
        echo "  Cache port 20100: LISTENING"
    else
        echo "  WARNING: Cache port 20100 not yet listening (may still be starting)"
    fi
else
    echo "  ERROR: Cache screen failed to start!"
    echo "  Check $LOGS_DIR/CACHE.log for errors"
fi
echo ""

# Use absolute path for specs
SPECS_DIR="$PROJECT_ROOT/specs/ethereum.json"
echo "Using static specs: $SPECS_DIR"

# Export RPC endpoint URLs as environment variables
# Set these before running the script:
#
# HTTP/HTTPS endpoints (required — at least 2, 3rd is optional):
#   export ETH_RPC_URL_1="https://mainnet.infura.io/v3/YOUR_INFURA_KEY"
#   export ETH_RPC_URL_2="https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY"
#   export ETH_RPC_URL_3="https://ethereum-rpc.publicnode.com"
#
# WebSocket endpoints (required for subscription support):
#   export ETH_WS_URL_1="wss://mainnet.infura.io/ws/v3/YOUR_INFURA_KEY"
#   export ETH_WS_URL_2="wss://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY"
#   export ETH_WS_URL_3="wss://ethereum-rpc.publicnode.com"

# Set defaults if not already exported (placeholders will fail to connect)
export ETH_RPC_URL_1="${ETH_RPC_URL_1:-https://eth.llamarpc.com}"
export ETH_RPC_URL_2="${ETH_RPC_URL_2:-https://json-rpc.8zfcse2amst1lajmh299uq4jn.blockchainnodeengine.com/?key=AIzaSyDyUtm6b-e-xKDQgVWzlroHdVTytiXEDik}"
export ETH_RPC_URL_3="${ETH_RPC_URL_3:-https://ethereum-rpc.publicnode.com}"
# Optional backup endpoint — emitted under `backup-direct-rpc:` only when set.
# Backup providers are consulted by the smart router only when every primary
# `direct-rpc` peer is exhausted (see consumer_session_manager.go backup fallback chain).
export ETH_RPC_URL_4="${ETH_RPC_URL_4:-}"

# WebSocket endpoints (required for subscriptions)
#   export ETH_WS_URL_1="wss://mainnet.infura.io/ws/v3/YOUR_INFURA_KEY"
#   export ETH_WS_URL_2="wss://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY"
#   export ETH_WS_URL_3="wss://ethereum-rpc.publicnode.com"
export ETH_WS_URL_1="${ETH_WS_URL_1:-wss://g.w.lavanet.xyz:443/gateway/eth/rpc/4926f3fd246058892909cdda0c88f8c7}"
export ETH_WS_URL_2="${ETH_WS_URL_2:-wss://g.w.lavanet.xyz:443/gateway/eth/rpc/4926f3fd246058892909cdda0c88f8c7}"
export ETH_WS_URL_3="${ETH_WS_URL_3:-wss://ethereum-rpc.publicnode.com}"

# Validate that real URLs are set (not placeholders)
for var_name in ETH_RPC_URL_1 ETH_RPC_URL_2 ETH_RPC_URL_3 ETH_WS_URL_1 ETH_WS_URL_2 ETH_WS_URL_3; do
    if [[ "${!var_name}" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
        echo "ERROR: $var_name contains placeholder!"
        echo ""
        echo "Set real Ethereum endpoints before running:"
        echo "  export ETH_RPC_URL_1='https://mainnet.infura.io/v3/YOUR_KEY'"
        echo "  export ETH_RPC_URL_2='https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY'"
        echo "  export ETH_RPC_URL_3='https://ethereum-rpc.publicnode.com'"
        echo "  export ETH_WS_URL_1='wss://mainnet.infura.io/ws/v3/YOUR_KEY'"
        echo "  export ETH_WS_URL_2='wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY'"
        echo "  export ETH_WS_URL_3='wss://ethereum-rpc.publicnode.com'"
        echo ""
        echo "Then run: $0"
        exit 1
    fi
done

# Generate smart router config only if missing or REGENERATE_CONFIG=1 (keeps manual edits otherwise)
if [[ -f "$CONFIG_FILE" && "$REGENERATE_CONFIG" != "1" ]]; then
    echo "Using existing config: $CONFIG_FILE"
    echo "  (Set REGENERATE_CONFIG=1 to regenerate from env vars)"
    echo ""
else
# Generate smart router config with DIRECT RPC (no providers!)
echo "Generating smart router config: $CONFIG_FILE"
echo ""
echo "Direct RPC Configuration:"
echo "  HTTP Endpoint 1: ${ETH_RPC_URL_1:0:50}..."
echo "  HTTP Endpoint 2: ${ETH_RPC_URL_2:0:50}..."
echo "  HTTP Endpoint 3: ${ETH_RPC_URL_3:0:50}..."
if [[ -n "$ETH_RPC_URL_4" ]]; then
    echo "  Backup Endpoint: ${ETH_RPC_URL_4:0:50}... (fallback-only)"
fi
echo ""
echo "  WebSocket Endpoints (Phase 5 - Subscriptions):"
echo "    WS Endpoint 1: ${ETH_WS_URL_1:0:50}..."
echo "    WS Endpoint 2: ${ETH_WS_URL_2:0:50}..."
echo "    WS Endpoint 3: ${ETH_WS_URL_3:0:50}..."
echo ""
echo "IMPORTANT: This is DIRECT RPC mode"
echo "    - Smart router connects DIRECTLY to Ethereum RPC endpoints"
echo "    - NO Lava providers in the middle!"
echo "    - 3 endpoints enable cross-validation testing (2-of-3 / 3-of-3)"
echo "    - Testing Phases 1-5 implementation"
echo ""

# Build the config file — each provider is a separate direct-rpc entry
cat > $CONFIG_FILE <<EOF
# Smart Router Direct RPC Configuration
# Testing Phases 1-5: JSON-RPC over HTTP/HTTPS + WebSocket Subscriptions
# Mode: Direct connections to Ethereum RPC endpoints (no Lava providers!)

endpoints:
  - listen-address: "0.0.0.0:3360"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    network-address: "0.0.0.0:3360"

direct-rpc:
  # HTTP Endpoint 1
  - name: "eth-rpc-1"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_1"
        skip-verifications:
          - chain-id
          - pruning

  # HTTP Endpoint 2
  - name: "eth-rpc-2"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_2"
        skip-verifications:
          - chain-id
          - pruning

  # HTTP Endpoint 3
  - name: "eth-rpc-3"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_3"
        skip-verifications:
          - chain-id
          - pruning

  # WebSocket Endpoint 1 (Phase 5 - Subscriptions)
  - name: "eth-ws-1"
    chain-id: "ETH1"
    api-interface: "websocket"
    node-urls:
      - url: "$ETH_WS_URL_1"

  # WebSocket Endpoint 2 (Phase 5 - Subscriptions)
  - name: "eth-ws-2"
    chain-id: "ETH1"
    api-interface: "websocket"
    node-urls:
      - url: "$ETH_WS_URL_2"

  # WebSocket Endpoint 3 (Phase 5 - Subscriptions)
  - name: "eth-ws-3"
    chain-id: "ETH1"
    api-interface: "websocket"
    node-urls:
      - url: "$ETH_WS_URL_3"
EOF

# Optional: emergency-fallback tier consulted only when every primary direct-rpc peer is exhausted.
if [[ -n "$ETH_RPC_URL_4" ]]; then
cat >> $CONFIG_FILE <<EOF

backup-direct-rpc:
  # Backup HTTP Endpoint (used only when all primary direct-rpc peers are exhausted)
  - name: "eth-rpc-backup"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_4"
        skip-verifications:
          - chain-id
          - pruning
EOF
fi

# Verify config file was created
echo ""
echo "Verifying generated config file..."
if [ -f "$CONFIG_FILE" ]; then
    FILE_SIZE=$(wc -c < "$CONFIG_FILE")
    echo "Smart router config exists: $CONFIG_FILE (size: $FILE_SIZE bytes)"
    echo ""
    echo "Config preview (first 30 lines):"
    head -n 30 "$CONFIG_FILE" | sed 's/^/  /'
    echo "  ..."
else
    echo "ERROR: Smart router config NOT found: $CONFIG_FILE"
    exit 1
fi
echo ""
fi  # end: regenerate config or use existing

# Determine which phases are active
PHASE_STATUS="Phases 1-3 (HTTP/HTTPS)"
if [[ "$WS_ENABLED" == "true" ]]; then
    PHASE_STATUS="Phases 1-5 (HTTP/HTTPS + WebSocket)"
fi

# Start Smart Router with DIRECT RPC (no providers!)
echo "[Test Setup] starting Smart Router (DIRECT RPC mode, standalone)"
echo ""
echo "Smart Router Configuration:"
echo "   - Mode: DIRECT RPC (bypasses Lava providers)"
echo "   - Protocols: JSON-RPC over HTTP/HTTPS"
if [[ "$WS_ENABLED" == "true" ]]; then
    echo "   - WebSocket: ENABLED (Phase 5 subscriptions)"
else
    echo "   - WebSocket: DISABLED (set ETH_WS_URL_1/2 to enable)"
fi
echo "   - HTTP Endpoints: 3 endpoints (parallel relay + cross-validation)"
echo "     Endpoint 1: ${ETH_RPC_URL_1:0:40}..."
echo "     Endpoint 2: ${ETH_RPC_URL_2:0:40}..."
echo "     Endpoint 3: ${ETH_RPC_URL_3:0:40}..."
if [[ -n "$ETH_RPC_URL_4" ]]; then
    echo "   - Backup:    1 endpoint (used only when all primaries exhausted)"
    echo "     Endpoint 4: ${ETH_RPC_URL_4:0:40}..."
fi
if [[ -n "$ETH_WS_URL_1" ]]; then
    echo "   - WS Endpoint 1: ${ETH_WS_URL_1:0:40}..."
fi
if [[ -n "$ETH_WS_URL_2" ]]; then
    echo "   - WS Endpoint 2: ${ETH_WS_URL_2:0:40}..."
fi
if [[ -n "$ETH_WS_URL_3" ]]; then
    echo "   - WS Endpoint 3: ${ETH_WS_URL_3:0:40}..."
fi
echo "   - Cache: Enabled (127.0.0.1:20100)"
echo "   - Specs: Static (no blockchain connection)"
echo "   - Listen: 0.0.0.0:3360"
echo ""

screen -d -m -S smartrouter bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcsmartrouter \
smartrouter_eth.yml \
--geolocation 1 \
--log-level debug \
--cache-be \"127.0.0.1:20100\" \
--use-static-spec $SPECS_DIR \
--skip-websocket-verification \
--metrics-listen-address ':7779' 2>&1 | tee $LOGS_DIR/SMARTROUTER.log" && sleep 0.25

sleep 3

# Verify smart router started successfully
echo "Verifying smart router screen session..."
if screen -list | grep -q "smartrouter"; then
    echo "Smart router screen is running"
else
    echo "ERROR: Smart router screen failed to start!"
    echo "  Check $LOGS_DIR/SMARTROUTER.log for errors"
    exit 1
fi
echo ""

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "Smart Router Direct RPC Setup Complete!"
echo "============================================"
echo "Cache:         127.0.0.1:20100 (metrics: 20200)"
echo "Smart Router:  0.0.0.0:3360 (metrics: 7779)"
echo ""
echo "Direct RPC Endpoints (Parallel Relay + Cross-Validation):"
echo "  HTTP 1: ${ETH_RPC_URL_1:0:50}..."
echo "  HTTP 2: ${ETH_RPC_URL_2:0:50}..."
echo "  HTTP 3: ${ETH_RPC_URL_3:0:50}..."
if [[ -n "$ETH_RPC_URL_4" ]]; then
    echo ""
    echo "Backup Endpoint (emergency fallback, not queried in parallel):"
    echo "  HTTP 4: ${ETH_RPC_URL_4:0:50}..."
fi
if [[ "$WS_ENABLED" == "true" ]]; then
    echo ""
    echo "WebSocket Endpoints (Subscriptions):"
    if [[ -n "$ETH_WS_URL_1" ]]; then
        echo "  WS 1: ${ETH_WS_URL_1:0:50}..."
    fi
    if [[ -n "$ETH_WS_URL_2" ]]; then
        echo "  WS 2: ${ETH_WS_URL_2:0:50}..."
    fi
    if [[ -n "$ETH_WS_URL_3" ]]; then
        echo "  WS 3: ${ETH_WS_URL_3:0:50}..."
    fi
fi
echo ""
echo "Parallel Relay: Requests sent to ALL 3 endpoints simultaneously"
echo "   First successful response wins (lower latency!)"
echo "   Cross-validation: use lava-cross-validation-* headers for consensus"
echo ""
echo "TESTING $PHASE_STATUS"
echo "  Phase 1: DirectRPCConnection foundation"
echo "  Phase 2: Session integration"
echo "  Phase 3: JSON-RPC relay logic"
echo "  Phase 4: REST relay logic"
if [[ "$WS_ENABLED" == "true" ]]; then
    echo "  Phase 5: WebSocket subscriptions"
fi
echo ""
echo "Test Commands (HTTP/JSON-RPC):"
echo "  # Get latest block number"
echo "  curl -X POST http://127.0.0.1:3360 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}'"
echo ""
echo "  # Get block by number"
echo "  curl -X POST http://127.0.0.1:3360 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBlockByNumber\",\"params\":[\"latest\",false],\"id\":1}'"
echo ""
echo "  # Get balance"
echo "  curl -X POST http://127.0.0.1:3360 \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"0xYOUR_ADDRESS\",\"latest\"],\"id\":1}'"
echo ""

if [[ "$WS_ENABLED" == "true" ]]; then
    echo "Test Commands (WebSocket Subscriptions - Phase 5):"
    echo "  # Install wscat if needed: npm install -g wscat"
    echo ""
    echo "  # Connect to WebSocket endpoint"
    echo "  wscat -c ws://127.0.0.1:3360/ws"
    echo ""
    echo "  # Once connected, subscribe to new blocks:"
    echo '  > {"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}'
    echo ""
    echo "  # Subscribe to pending transactions:"
    echo '  > {"jsonrpc":"2.0","id":2,"method":"eth_subscribe","params":["newPendingTransactions"]}'
    echo ""
    echo "  # Subscribe to logs (e.g., USDT transfers):"
    echo '  > {"jsonrpc":"2.0","id":3,"method":"eth_subscribe","params":["logs",{"address":"0xdAC17F958D2ee523a2206206994597C13D831ec7"}]}'
    echo ""
    echo "  # Unsubscribe (use subscription ID from response):"
    echo '  > {"jsonrpc":"2.0","id":4,"method":"eth_unsubscribe","params":["0xSUBSCRIPTION_ID"]}'
    echo ""
fi

echo "============================================"
echo "CROSS-VALIDATION TESTING (3 endpoints)"
echo "============================================"
echo ""
echo "  # 2-of-3 consensus (query 3 providers, 2 must agree):"
echo '  curl -v -X POST http://127.0.0.1:3360 \'
echo '    -H "Content-Type: application/json" \'
echo '    -H "lava-cross-validation-max-participants: 3" \'
echo '    -H "lava-cross-validation-agreement-threshold: 2" \'
echo '    -d '\''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'\'''
echo ""
echo "  # 3-of-3 strict consensus (all must agree):"
echo '  curl -v -X POST http://127.0.0.1:3360 \'
echo '    -H "Content-Type: application/json" \'
echo '    -H "lava-cross-validation-max-participants: 3" \'
echo '    -H "lava-cross-validation-agreement-threshold: 3" \'
echo '    -d '\''{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'\'''
echo ""
echo "  Response headers to check:"
echo "    lava-cross-validation-status          — consensus result"
echo "    lava-cross-validation-agreeing-providers — which providers agreed"
echo "    lava-cross-validation-all-providers    — all participants"
echo ""
echo "============================================"
echo "CACHE TESTING (use these to verify cache)"
echo "============================================"
echo ""
echo "Step 1: Send first request (expect CACHE MISS, then WRITE SUCCESS):"
echo '  curl -s -X POST http://127.0.0.1:3360 \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '\''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'\'''
echo ""
echo "Step 2: Send SAME request again (expect CACHE HIT):"
echo '  curl -s -X POST http://127.0.0.1:3360 \'
echo '    -H "Content-Type: application/json" \'
echo '    -d '\''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'\'''
echo ""
echo "Step 3: Force cache bypass (expect CACHE MISS even with cached data):"
echo '  curl -s -X POST http://127.0.0.1:3360 \'
echo '    -H "Content-Type: application/json" \'
echo '    -H "lava-force-cache-refresh: true" \'
echo '    -d '\''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'\'''
echo ""
echo "Monitor CACHE logs (look for HIT/MISS/WRITE):"
echo "  tail -f $LOGS_DIR/SMARTROUTER.log | grep -i 'CACHE'"
echo ""
echo "Expected log patterns:"
echo "  [CACHE] ✗ MISS - will relay to endpoint    <- First request"
echo "  [CACHE] ✓ WRITE SUCCESS - response cached  <- After relay"
echo "  [CACHE] ✓ HIT - returning cached response  <- Second request"
echo ""
echo "============================================"
echo ""
echo "Monitor Logs:"
echo "  tail -f $LOGS_DIR/SMARTROUTER.log | grep -i 'direct\\|endpoint\\|relay\\|subscription\\|CACHE'"
echo ""
echo "Metrics:"
echo "  Smart Router: http://localhost:7779/metrics"
echo "  Cache: http://localhost:20200/metrics"
echo ""
echo "What to Look For in Logs:"
echo "  - 'sending direct RPC request' (Phase 3 working!)"
echo "  - 'direct RPC request succeeded' (successful relay)"
echo "  - 'endpoint: infura-eth-mainnet' (using direct connections)"
echo "  - 'protocol: https' (HTTP protocol detection working)"
if [[ "$WS_ENABLED" == "true" ]]; then
    echo "  - 'DirectWS: subscription started' (Phase 5 working!)"
    echo "  - 'WebSocket pool: connection added' (connection pooling)"
    echo "  - 'DirectWS: client joined existing subscription' (deduplication)"
fi
echo ""
echo "To Stop All Services:"
echo "  killall lavap"
echo "  screen -wipe"
echo ""
echo "============================================"
echo "Ready to test!"
echo "============================================"
