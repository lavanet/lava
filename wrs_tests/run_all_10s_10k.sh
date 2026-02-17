#!/usr/bin/env bash
set -euo pipefail

cd /Users/anna/go/lava

WARMUP="${WARMUP:-100}"
MEASURED="${MEASURED:-10000}"
PARALLEL="${PARALLEL:-100}"
BUCKET_SEC="${BUCKET_SEC:-10}"

for d in test_latency test_sync test_availability test_stake; do
  echo "=== RUN ${d} (warmup=${WARMUP} measured=${MEASURED} parallel=${PARALLEL} bucket=${BUCKET_SEC}s) ==="
  python3 wrs_tests/_framework/analyze.py \
    --test-dir "wrs_tests/${d}" \
    --warmup "${WARMUP}" \
    --measured "${MEASURED}" \
    --parallel "${PARALLEL}" \
    --time-bucket-sec "${BUCKET_SEC}"
  echo "=== DONE ${d} ==="
done

