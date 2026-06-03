#!/bin/bash
# Integration test environment for majorityBaseline outlier detection validation.
#
# Based on init_lava_only_with_node_test_three_providers.sh with these changes:
# - Provider 3 points to a Python proxy instead of the real node
# - Consumer has --enable-periodic-probe-providers and --debug-probes enabled
# - Proxy has a control API for injecting block height overrides, latency, and errors
#
# Architecture:
#   Provider 1 --> Real Lava Node (127.0.0.1:1317, :26657)
#   Provider 2 --> Real Lava Node
#   Provider 3 --> Python Proxy (127.0.0.1:4000, :4010) --> Real Lava Node
#                      |
#                      +-- Control API: 127.0.0.1:4001
#
# Control the proxy:
#   curl -X POST http://127.0.0.1:4001/set-block?height=20000000   # contamination
#   curl -X POST http://127.0.0.1:4001/set-latency?ms=3000         # slow provider
#   curl -X POST http://127.0.0.1:4001/set-error?enabled=true      # kill provider
#   curl -X POST http://127.0.0.1:4001/reset                       # back to normal
#   curl http://127.0.0.1:4001/status                              # current config
#
# Monitor:
#   tail -f testutil/debugging/logs/CONSUMERS.log | grep -E "majorityBaseline|outlier|blocking|consensus"
#   curl http://127.0.0.1:7779/metrics | grep -E "outlier|consensus_failure|blocked"

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

LOGS_DIR=${__dir}/../../testutil/debugging/logs
PROXY_DIR=${__dir}/../../testutil/debugging/mock_node_proxy
mkdir -p $LOGS_DIR
rm -f $LOGS_DIR/*.log

killall screen
screen -wipe

echo "[Test Setup] installing all binaries"
make install-all

echo "[Test Setup] setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Test Setup] sleeping for node to finish setup"
sleep 5
wait_for_lava_node_to_start

GASPRICE="0.00002ulava"
specs="./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/testnet-2/specs/lava.json"
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
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

# Proxy ports for provider 3
PROXY_REST_PORT=4000
PROXY_RPC_PORT=4010
PROXY_CONTROL_PORT=4001

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --from servicer3  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;

sleep_until_next_epoch
wait_next_block

echo "[Changing Epoch Storage Params] submitting param change vote"
lavad tx gov submit-legacy-proposal param-change ./cookbook/param_changes/param_change_epoch_params.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
wait_next_block
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava;

# Start the Python proxy BEFORE providers
echo "[Test Setup] Starting Python proxy for provider 3"
screen -d -m -S proxy bash -c "python3 $PROXY_DIR/proxy.py \
--rest-port $PROXY_REST_PORT --rpc-port $PROXY_RPC_PORT --control-port $PROXY_CONTROL_PORT \
--target-rest http://127.0.0.1:1317 --target-rpc http://127.0.0.1:26657 \
2>&1 | tee $LOGS_DIR/PROXY.log" && sleep 1

# Providers 1 and 2 point to the real node
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER1_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ':7766' 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER2_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER2_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ':7756' 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

# Provider 3 points to the proxy for HTTP (ChainTracker block height) and real node for WebSocket (subscriptions)
screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER3_LISTENER LAV1 rest 'http://127.0.0.1:$PROXY_REST_PORT' \
$PROVIDER3_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:$PROXY_RPC_PORT,ws://127.0.0.1:26657/websocket' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ':7746' 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

wait_next_block

# Consumer with periodic probing and debug probes enabled
screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 --chain-id lava --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing \
--metrics-listen-address ':7779' --debug-probes --enable-periodic-probe-providers --majority-baseline-bucket-time-window 2m \
2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo ""
echo "=== Test Environment Ready ==="
echo ""
echo "Screens:"
screen -ls
echo ""
echo "Proxy control:"
echo "  curl -X POST http://127.0.0.1:$PROXY_CONTROL_PORT/set-block?height=20000000   # contaminate provider 3"
echo "  curl -X POST http://127.0.0.1:$PROXY_CONTROL_PORT/set-latency?ms=3000         # slow down provider 3"
echo "  curl -X POST http://127.0.0.1:$PROXY_CONTROL_PORT/set-error?enabled=true      # kill provider 3"
echo "  curl -X POST http://127.0.0.1:$PROXY_CONTROL_PORT/reset                       # restore provider 3"
echo "  curl http://127.0.0.1:$PROXY_CONTROL_PORT/status                              # check proxy state"
echo ""
echo "Monitor:"
echo "  tail -f $LOGS_DIR/CONSUMERS.log | grep -E 'majorityBaseline|outlier|blocking|consensus|latestSyncData'"
echo "  curl http://127.0.0.1:7779/metrics | grep -E 'outlier|consensus_failure|blocked'"
echo ""
echo "Send test relay:"
echo "  curl http://127.0.0.1:3360/cosmos/base/tendermint/v1beta1/blocks/latest"
echo ""
