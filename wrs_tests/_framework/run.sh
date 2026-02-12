#!/bin/bash
set -euo pipefail

# Shared runner invoked by per-test run.sh scripts.
# It expects the calling directory (test dir) to provide:
#   - configs/provider1.json, provider2.json, provider3.json
#   - configs/provider1.yml,  provider2.yml,  provider3.yml
#   - configs/consumer.yml
# and environment variables:
#   - WARMUP_REQUESTS (default 50)
#   - NUM_REQUESTS (default 500)
#   - PARALLELISM (default 1)

TEST_DIR="$(pwd)"
REPO_ROOT="$(cd "$TEST_DIR/../.." && pwd)"

WARMUP_REQUESTS="${WARMUP_REQUESTS:-50}"
NUM_REQUESTS="${NUM_REQUESTS:-500}"
PARALLELISM="${PARALLELISM:-1}"
IGNORE_CACHE="${IGNORE_CACHE:-true}"

LOGS_DIR="$TEST_DIR/outputs/logs"
mkdir -p "$LOGS_DIR"
rm -f "$LOGS_DIR"/*.log "$LOGS_DIR"/provider_hits.test3_simple.txt 2>/dev/null || true

echo "[wrs] repo=$REPO_ROOT"
echo "[wrs] test=$TEST_DIR"
echo "[wrs] warmup=$WARMUP_REQUESTS measured=$NUM_REQUESTS parallelism=$PARALLELISM ignore_cache=$IGNORE_CACHE"

cd "$REPO_ROOT"

# lavap config loading via viper expects config names in the configured search paths.
# Using relative paths from repo root works reliably; absolute paths do not.
REL_TEST_DIR="${TEST_DIR#$REPO_ROOT/}"
killall lavap 2>/dev/null || true
killall screen 2>/dev/null || true
sleep 2
screen -wipe 2>/dev/null || true

echo "[wrs] installing binaries"
make install-all

# Full static spec set is required (LAV1 imports other specs).
SPECS_DIR="./specs/mainnet-1/specs/tendermint.json,./specs/mainnet-1/specs/ibc.json,./specs/mainnet-1/specs/cosmossdk.json,./specs/testnet-2/specs/lava.json"

CONSUMER_CACHE_ARGS=""
if [ "$IGNORE_CACHE" = false ]; then
  screen -d -m -S cache bash -c "lavap cache 127.0.0.1:20100 --log_level debug"
  sleep 2
  CONSUMER_CACHE_ARGS="--cache-be 127.0.0.1:20100"
fi

FIRST_PROVIDER="provider-2220-tendermintrpc"

echo "[wrs] starting providers"
screen -d -m -S prov1 bash -c "cd $REPO_ROOT && lavap rpcprovider $REL_TEST_DIR/configs/provider1.yml --test_mode --test_responses $REL_TEST_DIR/configs/provider1.json --static-providers --use-static-spec $SPECS_DIR --geolocation 1 --log_level trace 2>&1 | tee $LOGS_DIR/PROVIDER1.log"

screen -d -m -S prov2 bash -c "cd $REPO_ROOT && lavap rpcprovider $REL_TEST_DIR/configs/provider2.yml --test_mode --test_responses $REL_TEST_DIR/configs/provider2.json --static-providers --use-static-spec $SPECS_DIR --geolocation 1 --log_level trace 2>&1 | tee $LOGS_DIR/PROVIDER2.log"

screen -d -m -S prov3 bash -c "cd $REPO_ROOT && lavap rpcprovider $REL_TEST_DIR/configs/provider3.yml --test_mode --test_responses $REL_TEST_DIR/configs/provider3.json --static-providers --use-static-spec $SPECS_DIR --geolocation 1 --log_level trace 2>&1 | tee $LOGS_DIR/PROVIDER3.log"

sleep 15

echo "[wrs] starting consumer"
AVAIL_WEIGHT="${AVAIL_WEIGHT:-0.3}"
LAT_WEIGHT="${LAT_WEIGHT:-0.3}"
SYNC_WEIGHT="${SYNC_WEIGHT:-0.2}"
STAKE_WEIGHT="${STAKE_WEIGHT:-0.2}"

screen -d -m -S consumer bash -c "cd $REPO_ROOT && lavap rpcsmartrouter $TEST_DIR/configs/consumer.yml --geolocation 1 --log_level trace --enable-selection-stats --concurrent-providers 1 $CONSUMER_CACHE_ARGS --allow-insecure-provider-dialing --use-static-spec $SPECS_DIR --provider-optimizer-availability-weight $AVAIL_WEIGHT --provider-optimizer-latency-weight $LAT_WEIGHT --provider-optimizer-sync-weight $SYNC_WEIGHT --provider-optimizer-stake-weight $STAKE_WEIGHT --probe-update-weight 0.0001 --metrics-listen-address :7779 --optimizer-qos-listen 2>&1 | tee $LOGS_DIR/CONSUMER.test3_simple.log"
screen -d -m -S consumer bash -c "cd $REPO_ROOT && lavap rpcsmartrouter $REL_TEST_DIR/configs/consumer.yml --geolocation 1 --log_level trace --enable-selection-stats --concurrent-providers 1 $CONSUMER_CACHE_ARGS --allow-insecure-provider-dialing --use-static-spec $SPECS_DIR --provider-optimizer-availability-weight $AVAIL_WEIGHT --provider-optimizer-latency-weight $LAT_WEIGHT --provider-optimizer-sync-weight $SYNC_WEIGHT --provider-optimizer-stake-weight $STAKE_WEIGHT --probe-update-weight 0.0001 --metrics-listen-address :7779 --optimizer-qos-listen 2>&1 | tee $LOGS_DIR/CONSUMER.test3_simple.log"

sleep 40

echo "[wrs] warmup $WARMUP_REQUESTS (first pinned to $FIRST_PROVIDER)"
for i in $(seq 1 "$WARMUP_REQUESTS"); do
  if [ "$i" = "1" ]; then
    curl -s -o /dev/null -X POST -H "Content-Type: application/json" -H "lava-select-provider: $FIRST_PROVIDER" \
      --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
      http://127.0.0.1:3361/1/lava/tendermintrpc/LAV1 2>/dev/null || true
  else
    curl -s -o /dev/null -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
      http://127.0.0.1:3361/1/lava/tendermintrpc/LAV1 2>/dev/null || true
  fi
done

echo "[wrs] measured $NUM_REQUESTS (parallelism=$PARALLELISM)"
PROVIDER_HITS_FILE="$LOGS_DIR/provider_hits.test3_simple.txt"
rm -f "$PROVIDER_HITS_FILE"
touch "$PROVIDER_HITS_FILE"

send_one_measured_request() {
  local headers provider
  headers=$(
    curl -s -D - -o /dev/null -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' \
      http://127.0.0.1:3361/1/lava/tendermintrpc/LAV1 2>/dev/null || true
  )
  provider=$(
    # If the consumer retried, the header can include a comma-separated list of providers.
    # We record ONLY the first provider (the initial selection) so the distribution reflects
    # optimizer choices rather than retry behavior.
    echo "$headers" | awk -F': ' 'tolower($1)=="lava-provider-address"{print $2}' | tr -d '\r' | awk -F',' '{gsub(/^[ \t]+|[ \t]+$/,"",$1); print $1}'
  )
  if [ -z "$provider" ]; then
    provider="(unknown)"
  fi
  echo "$provider"
}

if [ "$PARALLELISM" -gt 1 ]; then
  export -f send_one_measured_request
  seq 1 "$NUM_REQUESTS" | xargs -P "$PARALLELISM" -n 1 bash -c 'send_one_measured_request' >> "$PROVIDER_HITS_FILE"
else
  for i in $(seq 1 "$NUM_REQUESTS"); do
    send_one_measured_request >> "$PROVIDER_HITS_FILE"
  done
fi

echo ""
echo "[wrs] distribution:"
sort "$PROVIDER_HITS_FILE" | uniq -c | sort -nr | sed 's/^/  /'

