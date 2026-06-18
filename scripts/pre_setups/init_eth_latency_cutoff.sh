#!/bin/bash
# E2E setup for the per-chain latency cutoff (max-provider-latency).
#
# Stands up a dev lava node + 3 ETH1 providers, each backed by its own mock RPC
# server (scripts/mock_rpc_server) so we can inject latency per provider. The
# consumer is started with a config file carrying max-provider-latency.
#
# Modes (1st arg, default "cutoff"):
#   cutoff               provider3 slow, consumer cutoff = 1.0  -> provider3 put aside
#   regression-disabled  provider3 slow, consumer cutoff = 0    -> provider3 still selected (feature off)
#   regression-fallback  ALL providers slow, cutoff = 1.0       -> safety fallback keeps full pool
#
# After it is up, verify with: scripts/test/verify_latency_cutoff.sh <mode>
#
# Note: the mock delay is whole seconds (>=1). We use 2s with a 1.0s cutoff so a
# slow relay still succeeds (registers as high latency, not an availability
# failure). Keep the consumer relay timeout above 2s.

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh

MODE="${1:-cutoff}"
ROOT="$__dir/../.."
LOGS_DIR="$ROOT/testutil/debugging/logs"
mkdir -p "$LOGS_DIR"
rm -f "$LOGS_DIR"/*.log

GASPRICE="0.00002ulava"
PROVIDERSTAKE="500000000000ulava"

# mock backend ports and provider listeners (index 1..3)
MOCK_PORTS=(19991 19992 19993)
PROVIDER_LISTENERS=("127.0.0.1:2221" "127.0.0.1:2222" "127.0.0.1:2223")
PROVIDER_METRICS=(7776 7777 7778)
SERVICERS=(servicer1 servicer2 servicer3)

# cutoff value + which providers are slow, per mode
CUTOFF="1.0"
SLOW_INDEXES=(3)
case "$MODE" in
  cutoff)               CUTOFF="1.0"; SLOW_INDEXES=(3) ;;
  regression-disabled)  CUTOFF="0";   SLOW_INDEXES=(3) ;;
  regression-fallback)  CUTOFF="1.0"; SLOW_INDEXES=(1 2 3) ;;
  *) echo "unknown mode: $MODE (use cutoff | regression-disabled | regression-fallback)"; exit 1 ;;
esac
echo "[Setup] mode=$MODE  cutoff=$CUTOFF  slow_providers=${SLOW_INDEXES[*]}"

killall screen 2>/dev/null
screen -wipe 2>/dev/null

echo "[Setup] installing binaries"
make install-all

echo "[Setup] starting dev lava node"
screen -d -m -S node bash -c "$ROOT/scripts/start_env_dev.sh"
sleep 5
wait_for_lava_node_to_start

# The full spec bundle (get_all_specs) exceeds the gov proposal metadata limit, so we
# register only the closure needed for a provider/consumer on ETH1 in a single proposal:
# the lava chain spec LAV1 and its imports (COSMOSSDK -> TENDERMINT, IBC), plus ETH1.
# One proposal -> one vote (the 4s voting period makes multi-proposal loops racy).
echo "[Setup] proposing specs (LAV1+ETH1 closure)"
specs="specs/mainnet-1/specs/ibc.json,specs/mainnet-1/specs/tendermint.json,specs/mainnet-1/specs/cosmossdk.json,specs/testnet-2/specs/lava.json,specs/mainnet-1/specs/ethereum.json"
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 5

echo "[Setup] proposing plans"
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 5

echo "[Setup] subscription + staking 3 ETH1 providers"
lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
for i in 0 1 2; do
  lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "${PROVIDER_LISTENERS[$i]},1" 1 $(operator_address) -y \
    --from ${SERVICERS[$i]} --provider-moniker "${SERVICERS[$i]}" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
  wait_next_block
done

echo "[Setup] starting 3 mock backends"
for i in 0 1 2; do
  port=${MOCK_PORTS[$i]}
  screen -d -m -S mock$((i+1)) bash -c "cd $ROOT; go run ./scripts/mock_rpc_server -port $port 2>&1 | tee $LOGS_DIR/MOCK$((i+1)).log"
done
sleep 3

echo "[Setup] applying latency to slow providers: ${SLOW_INDEXES[*]}"
for idx in "${SLOW_INDEXES[@]}"; do
  port=${MOCK_PORTS[$((idx-1))]}
  # 2s sticky delay -> above the 1.0s cutoff, relay still succeeds
  curl -s "http://127.0.0.1:$port/control?wait=2&sticky=1" && echo " -> provider$idx (mock :$port) slowed to 2s"
done

sleep_until_next_epoch

# The mock RPC server is not a real eth node, so it cannot satisfy the ETH1 spec's
# provider-startup verifications (chain-id, pruning, historical-block distance, ...).
# We hand the provider a local copy of the ETH1 spec with those verifications stripped
# (--use-static-spec). The provider still proxies real relays to the mock (so the injected
# latency is preserved); only the unfakeable node validation is skipped. The consumer keeps
# using the on-chain ETH1 spec for pairing.
echo "[Setup] generating verification-stripped static spec"
STATIC_SPEC="$LOGS_DIR/eth1_static_spec.json"
python3 - "$ROOT/specs/mainnet-1/specs/ethereum.json" "$STATIC_SPEC" <<'PY'
import json, sys
src, dst = sys.argv[1], sys.argv[2]
d = json.load(open(src))
for s in d["proposal"]["specs"]:
    for ac in s.get("api_collections", []):
        ac["verifications"] = []
json.dump(d, open(dst, "w"))
PY

echo "[Setup] starting providers"
for i in 0 1 2; do
  # http-only node url + --skip-websocket-verification (the mock has no ws; eth_blockNumber
  # only uses http) + --use-static-spec to skip the node-validation the mock can't pass.
  screen -d -m -S provider$((i+1)) bash -c "source ~/.bashrc; lavap rpcprovider \
    ${PROVIDER_LISTENERS[$i]} ETH1 jsonrpc 'http://127.0.0.1:${MOCK_PORTS[$i]}' \
    --use-static-spec $STATIC_SPEC --skip-websocket-verification \
    --geolocation 1 --log_level debug --from ${SERVICERS[$i]} --chain-id lava \
    --metrics-listen-address \":${PROVIDER_METRICS[$i]}\" 2>&1 | tee $LOGS_DIR/PROVIDER$((i+1)).log" && sleep 0.25
done
sleep 8

# Select the committed consumer config that matches this mode's cutoff (these live in the
# config/ dir, which is a viper search path, and are passed by name). The cutoff value is
# the only thing that varies between modes, so two configs cover all three modes:
#   cutoff / regression-fallback -> 1.0 (eth_latency_cutoff_consumer.yml)
#   regression-disabled          -> 0   (eth_latency_cutoff_consumer_disabled.yml)
if [ "$CUTOFF" = "0" ]; then
  CONSUMER_CFG_NAME="eth_latency_cutoff_consumer_disabled.yml"
else
  CONSUMER_CFG_NAME="eth_latency_cutoff_consumer.yml"
fi

echo "[Setup] starting consumer (config: config/$CONSUMER_CFG_NAME, cutoff=$CUTOFF)"
screen -d -m -S consumer bash -c "source ~/.bashrc; cd $ROOT; lavap rpcconsumer config/$CONSUMER_CFG_NAME \
  --geolocation 1 --log_level debug --from user1 --chain-id lava \
  --allow-insecure-provider-dialing --metrics-listen-address \":7779\" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setup done (mode=$MODE) ---"
echo "consumer:        http://127.0.0.1:3333"
echo "consumer metrics: http://127.0.0.1:7779/metrics"
echo "verify with:     scripts/test/verify_latency_cutoff.sh $MODE"
screen -ls
