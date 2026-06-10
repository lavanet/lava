#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh

# GitHub variant of init_lava_only_with_node_three_providers.sh: instead of pointing the consumer at
# a local whitelist file, it publishes the per-run whitelist to a remote git repo and has the
# consumer fetch it with --providers-whitelist-config / --providers-whitelist-token.
#
# Nothing repo-specific is hardcoded -- configure it entirely through these environment variables:
#   PROVIDERS_WHITELIST_TOKEN  read-only token the CONSUMER uses to fetch the whitelist repo
#   PROVIDERS_WHITELIST_URL    whitelist directory URL, e.g. https://github.com/<owner>/<repo>/tree/main
#   WHITELIST_REPO_DIR         path to a local clone of that repo (the script writes + git-pushes here)
#
# Example:
#   export PROVIDERS_WHITELIST_TOKEN=<read_only_token>
#   export PROVIDERS_WHITELIST_URL=https://github.com/<owner>/<repo>/tree/main
#   export WHITELIST_REPO_DIR=/path/to/<repo>
#
# Fail fast if any are missing. In particular, without the token the consumer falls back to the
# (empty) --github-token, fetches unauthenticated, 404s on a private repo, and silently passes
# through (allowing ALL relays) -- which looks like success. Abort before the long node spin-up.
missing=""
[ -z "${PROVIDERS_WHITELIST_TOKEN}" ] && missing="$missing PROVIDERS_WHITELIST_TOKEN"
[ -z "${PROVIDERS_WHITELIST_URL}" ] && missing="$missing PROVIDERS_WHITELIST_URL"
[ -z "${WHITELIST_REPO_DIR}" ] && missing="$missing WHITELIST_REPO_DIR"
if [ -n "$missing" ]; then
    echo "ERROR: set these environment variables before running:$missing" >&2
    echo "  PROVIDERS_WHITELIST_TOKEN  read-only token the consumer uses to fetch the whitelist repo" >&2
    echo "  PROVIDERS_WHITELIST_URL    whitelist directory URL, e.g. https://github.com/<owner>/<repo>/tree/main" >&2
    echo "  WHITELIST_REPO_DIR         local clone of that repo (the script writes + pushes the list here)" >&2
    exit 1
fi

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

screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7766" 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25

screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER2_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER2_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ":7756" 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25

screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER3_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER3_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ":7746" 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25


wait_next_block

# Provider whitelist: build the list from the ACTUAL staked provider addresses, since keys are
# regenerated on every setup (a hardcoded list would exclude all providers next run). We allow
# servicer1 and servicer3 for LAV1 and intentionally leave servicer2 out, so the consumer is seen
# relaying only to the two whitelisted providers (servicer2 is filtered).
#
# The consumer fetches this list from the remote repo (PROVIDERS_WHITELIST_URL). Because addresses
# change every run, we write THIS run's addresses into the local clone (WHITELIST_REPO_DIR), commit,
# and push using your own git credentials -- NOT the read-only PROVIDERS_WHITELIST_TOKEN, which only
# lets the consumer read. Both are supplied via environment (validated at the top of this script).
PROVIDER_WHITELIST_FILE="${WHITELIST_REPO_DIR}/provider_whitelist.json"

if [ ! -d "${WHITELIST_REPO_DIR}/.git" ]; then
    echo "ERROR: WHITELIST_REPO_DIR ($WHITELIST_REPO_DIR) is not a git clone. Clone your whitelist repo there (the same repo PROVIDERS_WHITELIST_URL points at)." >&2
    exit 1
fi

WHITELIST_PROVIDER_1=$(lavad keys show servicer1 -a)
WHITELIST_PROVIDER_3=$(lavad keys show servicer3 -a)
# Schema is chain -> geolocation -> allowed providers. The providers and the consumer below all run
# with --geolocation 1 (USC), so servicer1/servicer3 are whitelisted under "USC".
cat > "$PROVIDER_WHITELIST_FILE" <<EOF
{
  "chains": {
    "LAV1": {
      "USC": ["$WHITELIST_PROVIDER_1", "$WHITELIST_PROVIDER_3"]
    }
  }
}
EOF
echo "[Provider Whitelist] wrote $PROVIDER_WHITELIST_FILE (allow servicer1=$WHITELIST_PROVIDER_1, servicer3=$WHITELIST_PROVIDER_3; servicer2 excluded)"

# Commit & push so the consumer can fetch it. Push the clone's current branch (must match the branch
# in PROVIDERS_WHITELIST_URL). Empty-commit guard: if addresses happen to be unchanged, `git commit`
# errors on "nothing to commit" -- tolerate that and still push.
WHITELIST_BRANCH=$(git -C "$WHITELIST_REPO_DIR" rev-parse --abbrev-ref HEAD)
git -C "$WHITELIST_REPO_DIR" add provider_whitelist.json
git -C "$WHITELIST_REPO_DIR" commit -m "test: provider whitelist for this run (servicer1, servicer3)" || echo "[Provider Whitelist] nothing to commit (addresses unchanged)"
if ! git -C "$WHITELIST_REPO_DIR" push origin "$WHITELIST_BRANCH"; then
    echo "ERROR: failed to push provider_whitelist.json to the whitelist repo; consumer would fetch a stale/missing list and pass through. Aborting." >&2
    exit 1
fi
echo "[Provider Whitelist] pushed to $PROVIDERS_WHITELIST_URL"
# Give GitHub's raw CDN a moment to serve the new commit before the consumer fetches it.
sleep 3

screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
--providers-whitelist-config '$PROVIDERS_WHITELIST_URL' --providers-whitelist-token '$PROVIDERS_WHITELIST_TOKEN' --providers-whitelist-refresh-interval 30s \
$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level trace --from user1 --chain-id lava --cache-be 127.0.0.1:20100 --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMERS.log" && sleep 0.25

echo "--- setting up screens done ---"
screen -ls

echo "Provider 1 command:"
echo "lavap rpcprovider $PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' $PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' $PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ':7766'"

echo "Provider 2 command:"
echo "lavap rpcprovider $PROVIDER2_LISTENER LAV1 rest '$LAVA_REST' $PROVIDER2_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' $PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --chain-id lava --metrics-listen-address ':7756'"

echo "Provider 3 command:"
echo "lavap rpcprovider $PROVIDER3_LISTENER LAV1 rest '$LAVA_REST' $PROVIDER3_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' $PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --chain-id lava --metrics-listen-address ':7746'"
