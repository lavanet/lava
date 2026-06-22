#!/bin/bash
# Verifies the per-chain latency cutoff against a running env brought up by
# scripts/pre_setups/init_eth_latency_cutoff.sh.
#
# Usage: scripts/test/verify_latency_cutoff.sh [cutoff|regression-disabled|regression-fallback]
#
# Approach: warm up so the slow provider's QoS latency climbs above the cutoff,
# then snapshot the per-provider selection counter, send a measurement burst, and
# snapshot again. Assert on the deltas:
#   cutoff               -> slow provider gains ~0 selections; others keep growing
#   regression-disabled  -> slow provider still gains selections (feature off)
#   regression-fallback  -> all providers keep growing (fallback kept the pool)

MODE="${1:-cutoff}"
CONSUMER="http://127.0.0.1:3333"
METRICS="http://127.0.0.1:7779/metrics"
WARMUP=60
MEASURE=100
TOL=2   # allowed stray selections for a "put aside" provider

P1=$(lavad keys show servicer1 -a)
P2=$(lavad keys show servicer2 -a)
P3=$(lavad keys show servicer3 -a)

relay() {
  curl -s -X POST -H 'Content-Type: application/json' "$CONSUMER" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' >/dev/null
}

# selection count for a provider address (0 if it has no metric line yet)
count_for() {
  curl -s "$METRICS" \
    | awk -v p="$1" '$1 ~ /^lava_consumer_provider_selections/ && index($0, p) { print $NF; found=1 } END { if (!found) print 0 }' \
    | head -n1
}

echo "[verify] mode=$MODE  warmup=$WARMUP measure=$MEASURE"
echo "[verify] warming up..."
for i in $(seq 1 $WARMUP); do relay; done

b1=$(count_for "$P1"); b2=$(count_for "$P2"); b3=$(count_for "$P3")
echo "[verify] baseline selections: p1=$b1 p2=$b2 p3(slow)=$b3"

echo "[verify] measurement burst..."
for i in $(seq 1 $MEASURE); do relay; done

a1=$(count_for "$P1"); a2=$(count_for "$P2"); a3=$(count_for "$P3")
d1=$((a1 - b1)); d2=$((a2 - b2)); d3=$((a3 - b3))
echo "[verify] deltas: p1=+$d1 p2=+$d2 p3(slow)=+$d3"

pass=1
case "$MODE" in
  cutoff)
    # slow provider must be put aside; the two fast ones must absorb the traffic
    [ "$d3" -le "$TOL" ] || { echo "FAIL: slow provider still selected (+$d3 > $TOL)"; pass=0; }
    [ $((d1 + d2)) -gt 0 ] || { echo "FAIL: fast providers got no traffic"; pass=0; }
    ;;
  regression-disabled)
    # cutoff disabled -> slow provider still participates (default behavior intact)
    [ "$d3" -gt 0 ] || { echo "FAIL: slow provider was excluded with cutoff disabled"; pass=0; }
    ;;
  regression-fallback)
    # everyone slow -> safety fallback keeps the full pool, all keep being selected
    [ "$d1" -gt 0 ] && [ "$d2" -gt 0 ] && [ "$d3" -gt 0 ] || { echo "FAIL: fallback did not keep all providers"; pass=0; }
    ;;
  *) echo "unknown mode: $MODE"; exit 2 ;;
esac

if [ "$pass" -eq 1 ]; then echo "[verify] PASS ($MODE)"; exit 0; else echo "[verify] FAILED ($MODE)"; exit 1; fi
