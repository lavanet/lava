#!/usr/bin/env bash
set -euo pipefail

# Sends N tendermintrpc "health" JSON-RPC requests through the local rpcsmartrouter
# and prints provider selection distribution + selection-stats headers.

N="${1:-50}"
URL="${2:-http://127.0.0.1:3361}"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

echo "[load] sending $N requests to $URL (method=health)"

for i in $(seq 1 "$N"); do
  curl -s -i -X POST "$URL" \
    -H 'Content-Type: application/json' \
    -H 'lava-force-cache-refresh: true' \
    -d '{"jsonrpc":"2.0","id":1,"method":"health","params":[]}' >>"$tmp"
  echo "" >>"$tmp"
done

echo ""
echo "[distribution] Lava-Provider-Address counts:"
grep -i '^Lava-Provider-Address:' "$tmp" | awk -F': ' '{print $2}' | sort | uniq -c | sort -nr || true

echo ""
echo "[sample] first 5 lava-selection-stats headers:"
grep -i '^lava-selection-stats:' "$tmp" | head -5 || true

echo ""
echo "[sample] first 5 Provider-Latest-Block headers:"
grep -i '^Provider-Latest-Block:' "$tmp" | head -5 || true


