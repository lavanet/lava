#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

# Use absolute paths for logs
LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p "$LOGS_DIR"
LOGS_DIR=$(cd "$LOGS_DIR" && pwd)
rm "$LOGS_DIR"/*.log

# Save project root for later use
PROJECT_ROOT=$(cd "${__dir}"/../.. && pwd)

# Kill all lavap and lavad processes
killall lavap lavad 2>/dev/null || true
sleep 1

# Kill all screen sessions
killall screen 2>/dev/null || true
sleep 1
screen -wipe
sleep 1

# Clean up any old generated configs in project root
echo "Cleaning up old configs..."
rm -f "$PROJECT_ROOT/smartrouter_base.yml" 2>/dev/null || true

echo "[Test Setup] installing all binaries"
make install-all

# Start consumer cache service
echo "[Test Setup] starting consumer cache service"
screen -d -m -S cache bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE.log" && sleep 0.25

sleep 2

# Use absolute path for specs
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
#   export BASE_RPC_URL_4="https://fourth-base-endpoint.pro/YOUR_KEY"

# Set defaults if not already exported (placeholders will fail to connect)
export BASE_RPC_URL_1="${BASE_RPC_URL_1:-https://base-mainnet.infura.io/v3/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export BASE_RPC_URL_2="${BASE_RPC_URL_2:-https://base-endpoint.quiknode.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export BASE_RPC_URL_3="${BASE_RPC_URL_3:-https://another-base-endpoint.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"
export BASE_RPC_URL_4="${BASE_RPC_URL_4:-https://fourth-base-endpoint.pro/XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX}"

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
if [[ "$BASE_RPC_URL_4" == *"XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"* ]]; then
    echo "Warning: BASE_RPC_URL_4 contains placeholder. Set real values with:"
    echo "  export BASE_RPC_URL_4='https://fourth-base-endpoint.pro/YOUR_KEY'"
fi

# Generate smartrouter consumer config on-the-fly with direct-rpc pointing to actual RPC nodes
# direct-rpc   = primary providers (Infura HTTP, QuickNode HTTP, Endpoint 4)
# backup-direct-rpc = backup provider (QuickNode endpoint 3)
CONFIG_FILE="$PROJECT_ROOT/smartrouter_base.yml"
echo "Generating smartrouter config: $CONFIG_FILE"

cat > "$CONFIG_FILE" <<EOF
endpoints:
  - chain-id: BASE
    api-interface: jsonrpc
    network-address: 127.0.0.1:3360

direct-rpc:
  - name: primary-infura
    api-interface: jsonrpc
    chain-id: BASE
    node-urls:
      - url: $BASE_RPC_URL_1
      - url: $BASE_RPC_URL_1
        addons:
          - archive
  - name: primary-quicknode
    api-interface: jsonrpc
    chain-id: BASE
    node-urls:
      - url: $BASE_RPC_URL_2
      - url: $BASE_RPC_URL_2
        addons:
          - debug
      - url: $BASE_RPC_URL_2
        addons:
          - archive
      - url: $BASE_RPC_URL_2
        addons:
          - debug
          - archive
  - name: primary-endpoint-4
    api-interface: jsonrpc
    chain-id: BASE
    node-urls:
      - url: $BASE_RPC_URL_4
      - url: $BASE_RPC_URL_4
        addons:
          - debug
      - url: $BASE_RPC_URL_4
        addons:
          - archive
      - url: $BASE_RPC_URL_4
        addons:
          - debug
          - archive

backup-direct-rpc:
  - name: backup-quicknode-3
    api-interface: jsonrpc
    chain-id: BASE
    node-urls:
      - url: $BASE_RPC_URL_3
      - url: $BASE_RPC_URL_3
        addons:
          - debug
      - url: $BASE_RPC_URL_3
        addons:
          - archive
      - url: $BASE_RPC_URL_3
        addons:
          - debug
          - archive
EOF

echo "Smartrouter config generated successfully"
echo ""
echo "Config preview:"
cat "$CONFIG_FILE"
echo ""

# Start consumer (rpcsmartrouter - direct-rpc mode, connects to RPC nodes directly)
echo "[Test Setup] starting consumer (rpcsmartrouter direct-rpc mode)"
screen -d -m -S consumer bash -c "cd $PROJECT_ROOT && source ~/.bashrc; lavap rpcsmartrouter \
smartrouter_base \
--geolocation 1 --log_level trace \
--use-static-spec $SPECS_DIR \
--skip-websocket-verification \
--cache-be \"127.0.0.1:20100\" \
--metrics-listen-address ':7779' \
2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo ""
echo "============================================"
echo "Test Setup Complete (Direct RPC Mode)"
echo "============================================"
echo "Consumer Cache:  127.0.0.1:20100 (metrics: 20200)"
echo "Consumer:        127.0.0.1:3360 (rpcsmartrouter, metrics: 7779)"
echo ""
echo "Direct RPC Endpoints:"
echo "  Primary 1 (Infura):      ${BASE_RPC_URL_1:0:50}..."
echo "  Primary 2 (QuickNode):   ${BASE_RPC_URL_2:0:50}..."
echo "  Primary 3 (Endpoint 4):  ${BASE_RPC_URL_4:0:50}..."
echo "  Backup     (QuickNode):  ${BASE_RPC_URL_3:0:50}..."
echo ""
echo "No intermediate rpcprovider processes — smartrouter connects directly to RPC nodes."
echo "Using static specs: $SPECS_DIR"
echo "Config: $CONFIG_FILE"
echo "Logs: $LOGS_DIR"
echo "============================================"