#!/bin/bash

SMART_ROUTER="https://bnb-chain-jsonrpc.dev-smart-router.magmadevs.com:443"
BSC_MAINNET="https://bsc.blockpi.network/v1/rpc/2ae6dd2c3d1292ac77769964354104cdcc01d3f3"

# Enable verbose mode for first request if VERBOSE env var is set
VERBOSE_MODE=${VERBOSE:-0}

echo "========================================"
echo "BSC Block Lag Monitoring Tool"
echo "========================================"
echo "Smart Router: $SMART_ROUTER"
echo "BSC Mainnet:  $BSC_MAINNET"
echo ""
echo "Running initial connectivity test..."
echo ""

# Initial connectivity test
TEST_RESPONSE=$(curl -s -w "\nHTTP:%{http_code}" -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  $SMART_ROUTER 2>&1)
TEST_HTTP_CODE=$(echo "$TEST_RESPONSE" | grep "HTTP:" | cut -d: -f2)
TEST_BODY=$(echo "$TEST_RESPONSE" | grep -v "HTTP:")

echo "Initial Smart Router Test:"
echo "  HTTP Code: $TEST_HTTP_CODE"
echo "  Response Body: $TEST_BODY"
echo "  Body Length: $(echo "$TEST_BODY" | wc -c | tr -d ' ') bytes"
echo ""

if [ "$TEST_HTTP_CODE" != "200" ] || [ $(echo "$TEST_BODY" | wc -c | tr -d ' ') -lt 10 ]; then
    echo "⚠️  WARNING: Smart Router may not be functioning correctly!"
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

echo "Starting continuous monitoring... (Press Ctrl+C to stop)"
echo ""
printf "%-10s | %-12s | %-12s | %-8s | %-10s\n" "Time" "Smart Router" "BSC Mainnet" "Lag" "Status"
echo "-----------|--------------|--------------|----------|------------"

while true; do
    TIMESTAMP=$(date +"%H:%M:%S")

    # Fetch Smart Router block with comprehensive error capture
    SR_TEMP_FILE=$(mktemp)
    SR_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}\nTIME_TOTAL:%{time_total}\nTIME_CONNECT:%{time_connect}\nCONTENT_TYPE:%{content_type}" \
      -X POST \
      -H "Content-Type: application/json" \
      -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36" \
      -H "Accept: application/json" \
      --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
      $SMART_ROUTER 2>&1 | tee "$SR_TEMP_FILE")
    
    SR_HTTP_CODE=$(echo "$SR_RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
    SR_TIME_TOTAL=$(echo "$SR_RESPONSE" | grep "TIME_TOTAL:" | cut -d: -f2)
    SR_TIME_CONNECT=$(echo "$SR_RESPONSE" | grep "TIME_CONNECT:" | cut -d: -f2)
    SR_CONTENT_TYPE=$(echo "$SR_RESPONSE" | grep "CONTENT_TYPE:" | cut -d: -f2-)
    SR_BODY=$(echo "$SR_RESPONSE" | grep -v "HTTP_CODE:" | grep -v "TIME_TOTAL:" | grep -v "TIME_CONNECT:" | grep -v "CONTENT_TYPE:")
    SR_BODY_SIZE=$(echo "$SR_BODY" | wc -c | tr -d ' ')
    SR_HEX=$(echo "$SR_BODY" | jq -r '.result' 2>/dev/null)
    SR_ERROR=$(echo "$SR_BODY" | jq -r '.error.message' 2>/dev/null)
    SR_ERROR_CODE=$(echo "$SR_BODY" | jq -r '.error.code' 2>/dev/null)
    
    rm -f "$SR_TEMP_FILE"

    # Fetch BSC Mainnet block with error capture
    MAINNET_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST -H "Content-Type: application/json" \
      --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
      $BSC_MAINNET 2>&1)
    MAINNET_HTTP_CODE=$(echo "$MAINNET_RESPONSE" | tail -n1)
    MAINNET_BODY=$(echo "$MAINNET_RESPONSE" | sed '$d')
    MAINNET_HEX=$(echo "$MAINNET_BODY" | jq -r '.result' 2>/dev/null)
    MAINNET_ERROR=$(echo "$MAINNET_BODY" | jq -r '.error.message' 2>/dev/null)

    # Check if both requests succeeded
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
        # Detailed error reporting
        echo "$TIMESTAMP | ERROR: Could not fetch blocks"
        if [ -z "$SR_HEX" ] || [ "$SR_HEX" == "null" ]; then
            echo "  ├─ Smart Router: FAILED"
            echo "  │  ├─ Endpoint: $SMART_ROUTER"
            echo "  │  ├─ HTTP Code: $SR_HTTP_CODE"
            echo "  │  ├─ Response Time: ${SR_TIME_TOTAL}s (connect: ${SR_TIME_CONNECT}s)"
            echo "  │  ├─ Content-Type: $SR_CONTENT_TYPE"
            echo "  │  ├─ Body Size: $SR_BODY_SIZE bytes"
            
            # Detect Cloudflare errors
            if [[ "$SR_BODY" == *"error code: 1200"* ]] || [[ "$SR_BODY" == *"error code: 1015"* ]]; then
                echo "  │  ├─ 🚨 CLOUDFLARE RATE LIMIT DETECTED"
                echo "  │  ├─ Issue: Cloudflare is blocking requests before they reach Smart Router"
                echo "  │  ├─ Solution 1: Increase sleep interval (currently 3s)"
                echo "  │  ├─ Solution 2: Contact DevOps to whitelist your IP"
                echo "  │  └─ Solution 3: Use authenticated endpoint if available"
            elif [ "$SR_HTTP_CODE" = "503" ] && [[ "$SR_CONTENT_TYPE" == *"text/plain"* ]]; then
                echo "  │  ├─ 🚨 POSSIBLE PROXY/WAF BLOCKING"
                echo "  │  └─ This doesn't appear to be from Smart Router itself"
            fi
            
            if [ ! -z "$SR_ERROR" ] && [ "$SR_ERROR" != "null" ]; then
                echo "  │  ├─ JSON-RPC Error Code: $SR_ERROR_CODE"
                echo "  │  ├─ JSON-RPC Error Message: $SR_ERROR"
            fi
            
            if [ "$SR_BODY_SIZE" -lt 10 ]; then
                echo "  │  ├─ ⚠️  CRITICAL: Response body is essentially empty!"
                echo "  │  └─ Raw Response: '$SR_BODY'"
            else
                echo "  │  ├─ Response Body:"
                echo "$SR_BODY" | sed 's/^/  │     /'
            fi
        else
            echo "  ├─ Smart Router: OK (Block: $SR_HEX)"
        fi
        if [ -z "$MAINNET_HEX" ] || [ "$MAINNET_HEX" == "null" ]; then
            echo "  └─ BSC Mainnet: FAILED"
            echo "     ├─ HTTP Code: $MAINNET_HTTP_CODE"
            if [ ! -z "$MAINNET_ERROR" ] && [ "$MAINNET_ERROR" != "null" ]; then
                echo "     ├─ JSON-RPC Error: $MAINNET_ERROR"
            fi
            if [ -z "$MAINNET_BODY" ]; then
                echo "     ├─ Response: Empty/No response"
            else
                echo "     └─ Response: $MAINNET_BODY" | head -c 200
                echo "..."
            fi
        else
            echo "  └─ BSC Mainnet: OK (Block: $MAINNET_HEX)"
        fi
        echo ""
    fi

    sleep 3
done