#!/bin/bash

SMART_ROUTER="https://bnb-chain-jsonrpc.dev-smart-router.magmadevs.com:443"
BSC_MAINNET="https://bsc.blockpi.network/v1/rpc/2ae6dd2c3d1292ac77769964354104cdcc01d3f3"

echo "Monitoring BSC Block Lag - Press Ctrl+C to stop"
echo ""
printf "%-10s | %-12s | %-12s | %-8s | %-10s\n" "Time" "Smart Router" "BSC Mainnet" "Lag" "Status"
echo "-----------|--------------|--------------|----------|------------"

while true; do
    TIMESTAMP=$(date +"%H:%M:%S")

    SR_HEX=$(curl -s -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
      $SMART_ROUTER | jq -r '.result' 2>/dev/null)

    MAINNET_HEX=$(curl -s -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
      $BSC_MAINNET | jq -r '.result' 2>/dev/null)

    if [ ! -z "$SR_HEX" ] && [ "$SR_HEX" != "null" ] && [ ! -z "$MAINNET_HEX" ] && [ "$MAINNET_HEX" != "null" ]; then
        SR=$(printf "%d" $SR_HEX 2>/dev/null)
        MAINNET=$(printf "%d" $MAINNET_HEX 2>/dev/null)
        LAG=$((MAINNET - SR))

        if [ $LAG -lt 0 ]; then
            STATUS="⚠️ AHEAD"
        elif [ $LAG -gt 100 ]; then
            STATUS="❌ CRITICAL"
        elif [ $LAG -gt 50 ]; then
            STATUS="❌ PROBLEM"
        elif [ $LAG -gt 20 ]; then
            STATUS="⚠️ MODERATE"
        elif [ $LAG -gt 10 ]; then
            STATUS="✅ OK"
        else
            STATUS="✅ EXCELLENT"
        fi

        printf "%-10s | %12d | %12d | %8d | %-10s\n" "$TIMESTAMP" "$SR" "$MAINNET" "$LAG" "$STATUS"
    else
        printf "%-10s | ERROR: Could not fetch blocks\n" "$TIMESTAMP"
    fi

    sleep 0
done