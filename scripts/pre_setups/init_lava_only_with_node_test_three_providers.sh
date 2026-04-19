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
echo "[Test Setup] waiting for node to start"
sleep 5
wait_for_lava_node_to_start

GASPRICE="0.00002ulava"

# Spec proposal — use lavad_tx_and_wait to ensure tx lands before voting
echo "[Test Setup] submitting spec proposal"
lavad_tx_and_wait tx gov submit-legacy-proposal spec-add ./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmoswasm.json,./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/mainnet-1/specs/cosmossdkv50.json,./specs/mainnet-1/specs/ethermint.json,./specs/mainnet-1/specs/ethereum.json,./specs/testnet-2/specs/lava.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad_tx_and_wait tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6

# Plans proposal
echo "[Test Setup] submitting plans proposal"
lavad_tx_and_wait tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad_tx_and_wait tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2220"
PROVIDER2_LISTENER="127.0.0.1:2221"
PROVIDER3_LISTENER="127.0.0.1:2222"

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --from servicer3  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;

sleep_until_next_epoch
wait_next_block

echo "[Test Setup] submitting param change proposal"
lavad_tx_and_wait tx gov submit-legacy-proposal param-change ./cookbook/param_changes/param_change_epoch_params.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad_tx_and_wait tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

echo "[Test Setup] verifying LAV1 spec is active"
lavad q spec show-chain-info LAV1 2>&1 | head -5

echo "[Test Setup] starting providers and consumer"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider --test_mode --test_responses ./scripts/test_data/test_responses_state_machine.json \
$PROVIDER1_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER1_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$PROVIDER1_LISTENER LAV1 grpc '127.0.0.1:9090' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7766" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider --test_mode --test_responses ./scripts/test_data/test_responses_state_machine.json \
$PROVIDER2_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER2_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$PROVIDER2_LISTENER LAV1 grpc '127.0.0.1:9090' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ":7756" 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider --test_mode --test_responses ./scripts/test_data/test_responses_state_machine.json \
$PROVIDER3_LISTENER LAV1 rest 'http://127.0.0.1:1317' \
$PROVIDER3_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:26657,ws://127.0.0.1:26657/websocket' \
$PROVIDER3_LISTENER LAV1 grpc '127.0.0.1:9090' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ":7746" 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25

sleep 5

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --debug-relays --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

sleep 10
echo ""
echo "=== Running smoke test ==="
echo "--- 10 REST requests ---"
for i in $(seq 1 10); do
  curl -s -o /dev/null -w "req $i: HTTP %{http_code} (%{time_total}s)\n" http://127.0.0.1:3360/cosmos/base/tendermint/v1beta1/blocks/latest
done

echo ""
echo "--- Policy decisions in consumer log ---"
grep -c 'policy.Decide' $LOGS_DIR/CONSUMERS.log && echo "policy.Decide calls found" || echo "no policy.Decide calls yet"
grep 'policy.Decide' $LOGS_DIR/CONSUMERS.log | tail -5

echo ""
echo "--- Error classification in consumer log ---"
grep 'non-retryable\|unsupported method\|node error detected' $LOGS_DIR/CONSUMERS.log | tail -5

echo ""
echo "=== Manual test commands ==="
echo "curl -s http://127.0.0.1:3360/cosmos/base/tendermint/v1beta1/blocks/latest | jq ."
echo "tail -f $LOGS_DIR/CONSUMERS.log | grep 'policy.Decide\|StateMachine\|retry'"
