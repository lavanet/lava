#!/bin/bash
# Usage: ./attack.sh [base|eth|mixed]  (default: all three sequentially)
set -e

TARGETS=${1:-"all"}
RESULTS="results"
mkdir -p "$RESULTS"

run() {
  local label=$1 target_file=$2 rate=$3 duration=$4
  local tmp
  tmp=$(mktemp /tmp/vegeta_XXXXXX.bin)
  echo ">>> [$label] rate=${rate}/s  duration=${duration}  targets=${target_file}"
  vegeta attack \
    -targets="$target_file" \
    -rate="$rate" \
    -duration="$duration" \
    -timeout=30s \
    -keepalive=true \
    -max-body=0 \
    > "$tmp"
  echo "    latencies:"
  vegeta report -type=text < "$tmp"
  echo "    histogram:"
  vegeta report -type=hist[0,10ms,50ms,100ms,500ms,1s,5s] < "$tmp"
  rm -f "$tmp"
  echo
}

# ─── Profiles ─────────────────────────────────────────────────────────────────
#
#  1. Warmup      —  5 req/s  ×  30s   (shakes out errors, checks reachability)
#  2. Sustained   — 20 req/s  × 120s   (steady baseline, watch p99)
#  3. Medium      — 75 req/s  ×  60s   (ramp: find throughput ceiling)
#  4. Burst       — 300 req/s ×  15s   (spike: how quickly does it recover?)
#  5. Heavy-only  — 10 req/s  ×  60s   (debug/trace only — max server CPU)

attack_base() {
  run "base_warmup"    targets_base.txt   5   30s
  run "base_sustained" targets_base.txt  20  120s
  run "base_medium"    targets_base.txt  75   60s
  run "base_burst"     targets_base.txt 300   15s
}

attack_eth() {
  run "eth_warmup"    targets_eth.txt   5   30s
  run "eth_sustained" targets_eth.txt  20  120s
  run "eth_medium"    targets_eth.txt  75   60s
  run "eth_burst"     targets_eth.txt 300   15s
}

attack_sol() {
  run "sol_warmup"    targets_sol.txt   5   30s
  run "sol_sustained" targets_sol.txt  20  120s
  run "sol_medium"    targets_sol.txt  75   60s
  run "sol_burst"     targets_sol.txt 300   15s
}

attack_mixed() {
  # Both endpoints at once, mixed methods
  run "mixed_warmup"    targets_mixed.txt   5   30s
  run "mixed_sustained" targets_mixed.txt  20  120s
  run "mixed_medium"    targets_mixed.txt  75   60s
  run "mixed_burst"     targets_mixed.txt 300   15s
}

attack_ramp() {
  local rates=(1 10 25 50 75 100 125 150 175 200)
  for rate in "${rates[@]}"; do
    run "ramp_${rate}rps" targets_mixed.txt "$rate" 60s
  done
  run "spike_500rps"  targets_mixed.txt 500  60s
  run "spike_1000rps" targets_mixed.txt 1000 30s
}

attack_heavy() {
  # Build a targets file with only debug/trace methods against both endpoints
  local tf="targets_heavy.txt"
  BASE_URL="https://base-jsonrpc.nadav.magmadevs.com:443"
  ETH_URL="https://eth-jsonrpc.nadav.magmadevs.com:443"
  for url in "$BASE_URL" "$ETH_URL"; do
    for body in \
      bodies/debug_traceBlockByNumber_call.json \
      bodies/debug_traceBlockByNumber_prestate.json \
      bodies/trace_block.json \
      bodies/trace_replayBlockTransactions.json \
      bodies/trace_replayBlockTransactions_vm.json
    do
      printf "POST %s\nContent-Type: application/json\n@%s\n\n" "$url" "$body"
    done
  done > "$tf"
  run "heavy_debug_trace" "$tf" 10 60s
}

# ─── Summary helper ───────────────────────────────────────────────────────────
summary() {
  echo "═══════════════════════════════════════"
  echo " SUMMARY (all captured runs)"
  echo "═══════════════════════════════════════"
  for f in "$RESULTS"/*.bin; do
    label=$(basename "$f" .bin)
    printf "\n── %s ──\n" "$label"
    vegeta report -type=text < "$f"
  done
}

# ─── Dispatch ─────────────────────────────────────────────────────────────────
case "$TARGETS" in
  base)   attack_base  ;;
  eth)    attack_eth   ;;
  sol)    attack_sol   ;;
  mixed)  attack_mixed ;;
  heavy)  attack_heavy ;;
  ramp)  attack_ramp  ;;
  all)
    attack_base
    attack_eth
    attack_mixed
    attack_heavy
    summary
    ;;
  *)
    echo "Usage: $0 [base|eth|mixed|heavy|all]"
    exit 1
    ;;
esac
