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
specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/testnet-2/specs/avalanche_p.json"
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

PROVIDER1_LISTENER="127.0.0.1:2221"

if [ -z "$AVA_BASE_URL" ]; then
  echo "ERROR: Avalanche P-Chain endpoint is empty. Set AVA_BASE_URL in scripts/vars/variables.sh or environment. for example, 'api.avax-test.network/ext/bc'"
  exit 1
fi

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "AVALANCHEPT" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "dummyMoniker"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

# Generate provider YAML config with explicit internal-path mapping.
# The AVALANCHEP spec requires internal-path "/P" for its API collections.
# CLI args format doesn't support internal-path, so we use a YAML config.
PROVIDER1_YAML="config/provider1_avaxp.yml"
mkdir -p config
cat > "$PROVIDER1_YAML" <<EOF
endpoints:
  - api-interface: jsonrpc
    chain-id: AVALANCHEPT
    network-address:
      address: $PROVIDER1_LISTENER
    node-urls:
      - url: https://${AVA_BASE_URL}/P
        internal-path: "/P"
EOF

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_YAML \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address \":7776\" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 AVALANCHEPT jsonrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address \":7779\" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls
