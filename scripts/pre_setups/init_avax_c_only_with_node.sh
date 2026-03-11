#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

killall screen
screen -wipe

echo "[Test Setup] installing all binaries"
make install-all 

echo "[Test Setup] setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Test Setup] sleeping 20 seconds for node to finish setup (if its not enough increase timeout)"
sleep 5
wait_for_lava_node_to_start

GASPRICE="0.00002ulava"
specs=$(get_all_specs)
# Ensure avalanche C-chain spec is explicitly included for this setup.
if [[ "$specs" != *"specs/mainnet-1/specs/avalanche_c.json"* ]]; then
  specs="specs/mainnet-1/specs/avalanche_c.json,$specs"
fi
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

# Plans proposal
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2220"
AVAXC_NODE_URL="${AVA_C_RPC}"

if [ -z "$AVAXC_NODE_URL" ]; then
  echo "ERROR: Avalanche C-Chain endpoint is empty. Set AVA_C_RPC in scripts/vars/variables.sh or environment."
  exit 1
fi

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "AVAXC" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

# Generate provider YAML config with explicit internal-path mapping.
# The AVAXC spec requires internal-path "/C/rpc" for its API collections.
# CLI args format doesn't support internal-path, so we use a YAML config.
# Viper searches for configs relative to CWD and ./config, so we write there.
PROVIDER1_YAML="config/provider1_avaxc.yml"
cat > "$PROVIDER1_YAML" <<EOF
endpoints:
  - api-interface: jsonrpc
    chain-id: AVAXC
    network-address:
      address: $PROVIDER1_LISTENER
    node-urls:
      - url: ${AVAXC_NODE_URL}/C/rpc
        internal-path: "/C/rpc"
EOF

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_YAML \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --skip-websocket-verification --metrics-listen-address \":7776\" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 AVAXC jsonrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address \":7779\" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls
