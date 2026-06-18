#!/bin/bash
# One-command end-to-end runner for the per-chain latency cutoff.
#
# Brings up the full environment (chain + mocks + providers + consumer), waits for
# it to be ready, runs the verification, then tears everything down. Wraps the two
# reusable scripts:
#   scripts/pre_setups/init_eth_latency_cutoff.sh   (env)
#   scripts/test/verify_latency_cutoff.sh           (assertions)
#
# Usage:
#   scripts/test/e2e_latency_cutoff.sh [cutoff|regression-disabled|regression-fallback|all]
#   scripts/test/e2e_latency_cutoff.sh cutoff --keep    # leave the env running afterwards
#
# Exit code is non-zero if any mode's verification fails.

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
ROOT="$__dir/../.."
INIT="$ROOT/scripts/pre_setups/init_eth_latency_cutoff.sh"
VERIFY="$ROOT/scripts/test/verify_latency_cutoff.sh"
SETUP_LOG="/tmp/lava_latency_e2e_setup.log"

ARG="${1:-cutoff}"
KEEP=0
[ "$2" = "--keep" ] && KEEP=1

if [ "$ARG" = "all" ]; then
  MODES=(cutoff regression-disabled regression-fallback)
else
  MODES=("$ARG")
fi

teardown() {
  screen -ls 2>/dev/null | grep -oE '[0-9]+\.(node|provider[0-9]|consumer|mock[0-9])' | cut -d. -f1 \
    | xargs -I{} screen -S {} -X quit 2>/dev/null
  killall lavad lavap 2>/dev/null
  pkill -f mock_rpc_server 2>/dev/null
  sleep 2
  screen -wipe >/dev/null 2>&1
}

wait_for_consumer() {
  # wait until setup finished, then until the consumer actually serves a relay
  local tries=0
  echo "[e2e] waiting for setup to finish..."
  until grep -q "setup done" "$SETUP_LOG" 2>/dev/null; do
    sleep 5; tries=$((tries+1))
    [ $tries -gt 180 ] && { echo "[e2e] setup timed out (no 'setup done' in $SETUP_LOG)"; tail -5 "$SETUP_LOG"; return 1; }
  done
  echo "[e2e] setup done; waiting for consumer to serve a relay (a fresh chain may need an epoch for pairing)..."
  tries=0
  local resp=""
  # ~6 min: fresh-setup pairing/probing can take an epoch or two
  while [ $tries -le 120 ]; do
    resp=$(curl -s -m 6 -X POST http://127.0.0.1:3333 \
           -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' 2>/dev/null)
    echo "$resp" | grep -q result && { echo "[e2e] consumer serving."; return 0; }
    sleep 3; tries=$((tries+1))
    [ $((tries % 10)) -eq 0 ] && echo "[e2e]   still waiting (${tries}/120), last reply: ${resp:0:80}"
  done
  echo "[e2e] consumer never served a relay. last reply: ${resp:0:200}"
  echo "[e2e] consumer log tail:"; tail -6 "$ROOT/testutil/debugging/logs/CONSUMERS.log" 2>/dev/null | sed 's/StackTrace.*//'
  return 1
}

overall=0
for mode in "${MODES[@]}"; do
  echo "============================================================"
  echo "[e2e] mode=$mode : bringing up environment"
  echo "============================================================"
  teardown
  bash "$INIT" "$mode" > "$SETUP_LOG" 2>&1
  if ! wait_for_consumer; then
    echo "[e2e] FAILED ($mode): environment did not come up (see $SETUP_LOG)"
    overall=1
    continue
  fi
  echo "[e2e] environment ready, running verification"
  if bash "$VERIFY" "$mode"; then
    echo "[e2e] RESULT $mode: PASS"
  else
    echo "[e2e] RESULT $mode: FAIL"
    overall=1
  fi
done

if [ "$KEEP" -eq 1 ]; then
  echo "[e2e] leaving environment running (--keep). Tear down with: killall lavad lavap; pkill -f mock_rpc_server"
else
  echo "[e2e] tearing down"
  teardown
fi

[ "$overall" -eq 0 ] && echo "[e2e] ALL PASSED" || echo "[e2e] SOME FAILED"
exit $overall
