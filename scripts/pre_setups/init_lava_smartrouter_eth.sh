#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

# Use absolute paths for logs
LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
LOGS_DIR=$(cd "$LOGS_DIR" && pwd)
rm $LOGS_DIR/*.log 2>/dev/null || true

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

# Clean up any old generated configs in project root
echo "Cleaning up old smart router configs..."
rm -f $PROJECT_ROOT/smartrouter_eth.yml 2>/dev/null || true

echo "============================================"
echo "Smart Router Direct RPC Test Setup"
echo "============================================"
echo "Testing: Phases 1-3 (JSON-RPC over HTTP/HTTPS)"
echo "Mode: DIRECT RPC (no providers!)"
echo "============================================"
echo ""

echo "[Test Setup] installing all binaries"
make install-all 

# Start cache services (optional for smart router)
echo "[Test Setup] starting smart router cache service"
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

sleep 2

# Use absolute path for specs
SPECS_DIR="$PROJECT_ROOT/specs/mainnet-1/specs/ethereum.json"
echo "Using static specs: $SPECS_DIR"

# Export RPC endpoint URLs as environment variables
# Set these before running the script:
#   export ETH_RPC_URL_1="https://mainnet.infura.io/v3/YOUR_INFURA_KEY"
#   export ETH_RPC_URL_2="https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY"
#   
# For testing Phase 3 (JSON-RPC only), we only need HTTPS endpoints

# Set defaults if not already exported (placeholders will fail to connect)
export ETH_RPC_URL_1="${ETH_RPC_URL_1:-https://mainnet.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export ETH_RPC_URL_2="${ETH_RPC_URL_2:-https://eth-mainnet.g.alchemy.com/v2/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"

# Validate that real URLs are set (not placeholders)
if [[ "$ETH_RPC_URL_1" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "❌ ERROR: ETH_RPC_URL_1 contains placeholder!"
    echo ""
    echo "Set real Ethereum RPC endpoints before running:"
    echo "  export ETH_RPC_URL_1='https://mainnet.infura.io/v3/YOUR_INFURA_KEY'"
    echo "  export ETH_RPC_URL_2='https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY'"
    echo ""
    echo "Then run: $0"
    exit 1
fi

if [[ "$ETH_RPC_URL_2" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "❌ ERROR: ETH_RPC_URL_2 contains placeholder!"
    echo ""
    echo "Set real Ethereum RPC endpoints before running:"
    echo "  export ETH_RPC_URL_1='https://mainnet.infura.io/v3/YOUR_INFURA_KEY'"
    echo "  export ETH_RPC_URL_2='https://eth-mainnet.g.alchemy.com/v2/YOUR_ALCHEMY_KEY'"
    echo ""
    echo "Then run: $0"
    exit 1
fi

# Generate smart router config with DIRECT RPC (no providers!)
CONFIG_FILE="$PROJECT_ROOT/smartrouter_eth.yml"
echo "Generating smart router config: $CONFIG_FILE"
echo ""
echo "Direct RPC Configuration:"
echo "  Endpoint 1 (Infura):  ${ETH_RPC_URL_1:0:50}..."
echo "  Endpoint 2 (Alchemy): ${ETH_RPC_URL_2:0:50}..."
echo ""
echo "⚠️  IMPORTANT: This is DIRECT RPC mode"
echo "    - Smart router connects DIRECTLY to Ethereum RPC endpoints"
echo "    - NO Lava providers in the middle!"
echo "    - Testing Phases 1-3 implementation"
echo ""

cat > $CONFIG_FILE <<EOF
# Smart Router Direct RPC Configuration
# Testing Phases 1-3: JSON-RPC over HTTP/HTTPS
# Mode: Direct connections to Ethereum RPC endpoints (no Lava providers!)

endpoints:
  - listen-address: "0.0.0.0:3360"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    network-address: "0.0.0.0:3360"  # Simple string format

# Static providers - DIRECT RPC mode (bypasses Lava provider-relay protocol)
static-providers:
  # Endpoint 1: Infura (primary)
  - name: "infura-eth-mainnet"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_1"
        addons:
          - archive  # Infura typically provides archive data
  
  # Endpoint 2: Second endpoint (parallel relay / backup)
  - name: "eth-endpoint-2"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "$ETH_RPC_URL_2"
        addons:
          - archive
          - debug
          - trace

# NOTE: WebSocket endpoints commented out (Phase 5 not implemented yet)
# Uncomment after Phase 5 implementation:
#
# static-providers:
#   - name: "infura-eth-websocket"
#     chain-id: "ETH1"
#     api-interface: "jsonrpc"
#     node-urls:
#       - url: "wss://mainnet.infura.io/ws/v3/YOUR_KEY"
#   
#   - name: "alchemy-eth-websocket"
#     chain-id: "ETH1"
#     api-interface: "jsonrpc"
#     node-urls:
#       - url: "wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY"
EOF

# Verify config file was created
echo ""
echo "Verifying generated config file..."
if [ -f "$CONFIG_FILE" ]; then
    FILE_SIZE=$(wc -c < "$CONFIG_FILE")
    echo "✓ Smart router config exists: $CONFIG_FILE (size: $FILE_SIZE bytes)"
    echo ""
    echo "Config preview (first 20 lines):"
    head -n 20 "$CONFIG_FILE" | sed 's/^/  /'
    echo "  ..."
else
    echo "✗ ERROR: Smart router config NOT found: $CONFIG_FILE"
    exit 1
fi
echo ""

# Start Smart Router with DIRECT RPC (no providers!)
echo "[Test Setup] starting Smart Router (DIRECT RPC mode, standalone)"
echo ""
echo "🚀 Smart Router Configuration:"
echo "   - Mode: DIRECT RPC (bypasses Lava providers)"
echo "   - Protocols: JSON-RPC over HTTP/HTTPS"
echo "   - Endpoints: 2 endpoints (parallel relay)"
echo "     • Infura: ${ETH_RPC_URL_1:0:40}..."
echo "     • Endpoint 2: ${ETH_RPC_URL_2:0:40}..."
echo "   - Cache: Enabled (127.0.0.1:20100)"
echo "   - Specs: Static (no blockchain connection)"
echo "   - Listen: 0.0.0.0:3360"
echo ""

screen -d -m -S smartrouter bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcsmartrouter \
smartrouter_eth.yml \
--geolocation 1 \
--log_level trace \
--cache-be \"127.0.0.1:20100\" \
--use-static-spec $SPECS_DIR \
--metrics-listen-address ':7779' 2>&1 | tee $LOGS_DIR/SMARTROUTER.log" && sleep 0.25

sleep 3

# Verify smart router started successfully
echo "Verifying smart router screen session..."
if screen -list | grep -q "smartrouter"; then
    echo "✓ Smart router screen is running"
else
    echo "✗ ERROR: Smart router screen failed to start!"
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
echo "Direct RPC Endpoints (Parallel Relay):"
echo "  1. Infura:     ${ETH_RPC_URL_1:0:50}..."
echo "  2. Endpoint 2: ${ETH_RPC_URL_2:0:50}..."
echo ""
echo "⚡ Parallel Relay: Requests sent to BOTH endpoints simultaneously"
echo "   First successful response wins (lower latency!)"
echo ""
echo "⚡ TESTING PHASES 1-3 ⚡"
echo "  ✅ Phase 1: DirectRPCConnection foundation"
echo "  ✅ Phase 2: Session integration" 
echo "  ✅ Phase 3: JSON-RPC relay logic"
echo ""
echo "🔬 Test Commands:"
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
echo "📊 Monitor Logs:"
echo "  tail -f $LOGS_DIR/SMARTROUTER.log | grep -i 'direct.*rpc\\|endpoint\\|relay'"
echo ""
echo "📈 Metrics:"
echo "  Smart Router: http://localhost:7779/metrics"
echo "  Cache: http://localhost:20200/metrics"
echo ""
echo "🔍 What to Look For in Logs:"
echo "  - 'sending direct RPC request' (Phase 3 working!)"
echo "  - 'direct RPC request succeeded' (successful relay)"
echo "  - 'endpoint: infura-eth-mainnet' or 'alchemy-eth-mainnet' (using direct connections)"
echo "  - 'protocol: https' (HTTP protocol detection working)"
echo "  - 'IsDirectRPC()' or 'DirectRPCSessionConnection' (session management working)"
echo ""
echo "✋ To Stop All Services:"
echo "  killall lavap"
echo "  screen -wipe"
echo ""
echo "============================================"
echo "Ready to test! 🚀"
echo "============================================"
