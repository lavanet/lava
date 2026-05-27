#!/bin/bash
# init_lava_chaos_test_env.sh
#
# Pre-release chaos-testing variant of init_lava_only_with_node_three_providers_shared_state.sh.
# Wires testutil/debugging/mock_node_proxy/proxy.py between each provider and the lavad
# node so test scenarios can inject block-height overrides / latency / errors per provider
# at runtime via each proxy's HTTP control API. See:
#   protocol/rpcconsumer/docs/probing/improvements-proposals/pre-release-testing-infrastructure.md
#
# Differences from the shared-state script (in addition to its differences from the base):
#   1. Three proxy.py instances, one per provider. Each proxy fronts both REST and Tendermint
#      RPC; each has its own control port. WebSocket subscriptions still go directly to lavad
#      (proxy.py is HTTP-only) — note that Tendermint subscriptions bypass the chaos layer.
#   2. Provider command lines point at the proxies, not at $LAVA_REST / $LAVA_RPC directly.
#
# Inherits from the shared-state script:
#   - Two rpcconsumer processes (consumers_1 and consumers_2), both with --shared-state
#     pointing at the same cache-be (127.0.0.1:20100).
#   - --enable-periodic-probe-providers + --debug-probes +
#     --majority-baseline-bucket-time-window 2m on both consumers.
#   - LAV1 spec (testnet-2/specs/lava.json) with local lavad URLs.
#   - Both consumers run as --from user1; per-curl identity via the `dapp-id` HTTP header.

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
# Minimal hardcoded list — get_all_specs has caused submission failures (proposal
# silently doesn't commit) so we use the same fixed list as the pre-update base script.
specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/cosmoswasm.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/mainnet-1/specs/lava-mainnet.json"
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

# Per-provider proxy.py ports. Each proxy fronts the lavad node for one provider so test
# scenarios can independently inject failure modes ("provider 1 lies, providers 2+3 honest").
# REST + RPC are the data ports the provider points at; CTRL is the proxy's HTTP control API.
PROXY1_REST=4000 ; PROXY1_RPC=4010 ; PROXY1_CTRL=4001
PROXY2_REST=4100 ; PROXY2_RPC=4110 ; PROXY2_CTRL=4101
PROXY3_REST=4200 ; PROXY3_RPC=4210 ; PROXY3_CTRL=4201

lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --from servicer3  --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;


sleep_until_next_epoch
wait_next_block

echo "[Chaning Epoch Storage Params] submitting param change vote"
lavad tx gov submit-legacy-proposal param-change ./cookbook/param_changes/param_change_epoch_params.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE;
wait_next_block
wait_next_block
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava;

screen -d -m -S cache_consumer bash -c "source ~/.bashrc; lavap cache \
127.0.0.1:20100 --metrics_address 0.0.0.0:20200 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE_CONSUMER.log" && sleep 0.25
sleep 2;

# proxy.py instances — one per provider. HTTP-only (REST + Tendermint RPC); WebSocket
# subscriptions still talk to lavad directly. Each proxy's control port lets test scenarios
# inject block overrides / latency / errors per provider at runtime. See
# testutil/debugging/mock_node_proxy/proxy.py for the control API.
screen -d -m -S proxy1 bash -c "python3 testutil/debugging/mock_node_proxy/proxy.py \
--rest-port $PROXY1_REST --rpc-port $PROXY1_RPC --control-port $PROXY1_CTRL \
--target-rest $LAVA_REST --target-rpc $LAVA_RPC 2>&1 | tee $LOGS_DIR/PROXY1.log" && sleep 0.25
screen -d -m -S proxy2 bash -c "python3 testutil/debugging/mock_node_proxy/proxy.py \
--rest-port $PROXY2_REST --rpc-port $PROXY2_RPC --control-port $PROXY2_CTRL \
--target-rest $LAVA_REST --target-rpc $LAVA_RPC 2>&1 | tee $LOGS_DIR/PROXY2.log" && sleep 0.25
screen -d -m -S proxy3 bash -c "python3 testutil/debugging/mock_node_proxy/proxy.py \
--rest-port $PROXY3_REST --rpc-port $PROXY3_RPC --control-port $PROXY3_CTRL \
--target-rest $LAVA_REST --target-rpc $LAVA_RPC 2>&1 | tee $LOGS_DIR/PROXY3.log" && sleep 0.25
sleep 2;

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest 'http://127.0.0.1:$PROXY1_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:$PROXY1_RPC,$LAVA_RPC_WS' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7766" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER2_LISTENER LAV1 rest 'http://127.0.0.1:$PROXY2_REST' \
$PROVIDER2_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:$PROXY2_RPC,$LAVA_RPC_WS' \
$PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ":7756" 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER3_LISTENER LAV1 rest 'http://127.0.0.1:$PROXY3_REST' \
$PROVIDER3_LISTENER LAV1 tendermintrpc 'http://127.0.0.1:$PROXY3_RPC,$LAVA_RPC_WS' \
$PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ":7746" 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25


wait_next_block

# Consumer 1 — listens on :3360 (rest), :3361 (tendermintrpc), :3362 (grpc), metrics :7779
# --shared-state and --cache-be wire this consumer into the shared cache backend.
# --log_level trace exposes the [Optimizer] returned providers line (LavaFormatTrace).
screen -d -m -S consumers_1 bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --from user1 --chain-id lava --shared-state --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing --metrics-listen-address \":7779\" --debug-probes --enable-periodic-probe-providers --majority-baseline-bucket-time-window 2m 2>&1 | tee $LOGS_DIR/CONSUMERS_1.log" && sleep 0.25

# Consumer 2 — listens on :3460 (rest), :3461 (tendermintrpc), :3462 (grpc), metrics :7780
# Same --from user1; per-request identity (dappId) is set via the dapp-id header
# on the client, not via the staking key, so one staked user covers the demo.
screen -d -m -S consumers_2 bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3460 LAV1 rest 127.0.0.1:3461 LAV1 tendermintrpc 127.0.0.1:3462 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --from user1 --chain-id lava --shared-state --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing --metrics-listen-address \":7780\" --debug-probes --enable-periodic-probe-providers --majority-baseline-bucket-time-window 2m 2>&1 | tee $LOGS_DIR/CONSUMERS_2.log" && sleep 0.25


echo "--- setting up screens done ---"
screen -ls

echo ""
echo "=========================================================================="
echo "Process layout:"
echo "  lava node      : screen -r node"
echo "  cache server   : screen -r cache_consumer  (127.0.0.1:20100, metrics :20200)"
echo "  proxy 1        : screen -r proxy1           (rest :$PROXY1_REST, rpc :$PROXY1_RPC, ctrl :$PROXY1_CTRL)"
echo "  proxy 2        : screen -r proxy2           (rest :$PROXY2_REST, rpc :$PROXY2_RPC, ctrl :$PROXY2_CTRL)"
echo "  proxy 3        : screen -r proxy3           (rest :$PROXY3_REST, rpc :$PROXY3_RPC, ctrl :$PROXY3_CTRL)"
echo "  provider 1     : screen -r provider1       ($PROVIDER1_LISTENER, metrics :7766)  -> via proxy 1"
echo "  provider 2     : screen -r provider2       ($PROVIDER2_LISTENER, metrics :7756)  -> via proxy 2"
echo "  provider 3     : screen -r provider3       ($PROVIDER3_LISTENER, metrics :7746)  -> via proxy 3"
echo "  consumer 1     : screen -r consumers_1     (rest :3360, tmrpc :3361, grpc :3362, metrics :7779)"
echo "  consumer 2     : screen -r consumers_2     (rest :3460, tmrpc :3461, grpc :3462, metrics :7780)"
echo ""
echo "Logs:"
echo "  $LOGS_DIR/{CACHE_CONSUMER,PROXY1,PROXY2,PROXY3,PROVIDER1,PROVIDER2,PROVIDER3,CONSUMERS_1,CONSUMERS_2}.log"
echo ""
echo "Smoke test (use either consumer):"
echo "  curl -s -H 'dapp-id: alice' http://127.0.0.1:3361/status | head -c 200"
echo "  curl -s -H 'dapp-id: alice' http://127.0.0.1:3461/status | head -c 200"
echo ""
echo "Proxy control APIs (chaos injection — see proxy.py docstring for full set):"
echo "  curl -s http://127.0.0.1:$PROXY1_CTRL/status                                  # show proxy 1 state"
echo "  curl -s -X POST http://127.0.0.1:$PROXY1_CTRL/set-block?height=20000000      # provider 1 lies"
echo "  curl -s -X POST http://127.0.0.1:$PROXY1_CTRL/set-latency?ms=3000            # provider 1 slow"
echo "  curl -s -X POST http://127.0.0.1:$PROXY1_CTRL/set-error?enabled=true         # provider 1 errors"
echo "  curl -s -X POST http://127.0.0.1:$PROXY1_CTRL/reset                          # back to transparent"
echo ""
echo "See pre-release-testing-infrastructure.md for the chaos-test orchestration."
echo "=========================================================================="
