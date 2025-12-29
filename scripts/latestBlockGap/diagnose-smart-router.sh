#!/bin/bash

SMART_ROUTER="https://bnb-chain-jsonrpc.dev-smart-router.magmadevs.com:443"

echo "========================================"
echo "Smart Router Diagnostic Tool"
echo "========================================"
echo "Target: $SMART_ROUTER"
echo ""
echo "Running comprehensive diagnostics..."
echo ""

# Test 1: DNS Resolution
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: DNS Resolution"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
HOST=$(echo "$SMART_ROUTER" | sed 's|https://||' | sed 's|:.*||')
echo "Hostname: $HOST"
DNS_RESULT=$(nslookup "$HOST" 2>&1)
echo "$DNS_RESULT"
echo ""

# Test 2: TCP Connection
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: TCP Connection (Port 443)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
nc -zv "$HOST" 443 2>&1 || echo "TCP connection failed!"
echo ""

# Test 3: SSL/TLS Handshake
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: SSL/TLS Certificate"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo | openssl s_client -connect "$HOST:443" -servername "$HOST" 2>/dev/null | openssl x509 -noout -dates -subject 2>/dev/null || echo "SSL handshake failed!"
echo ""

# Test 4: Basic HTTP GET
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 4: HTTP GET Request"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -v -X GET "$SMART_ROUTER" 2>&1 | head -30
echo ""

# Test 5: JSON-RPC POST with verbose output
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 5: JSON-RPC eth_blockNumber (VERBOSE)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -v -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  "$SMART_ROUTER" 2>&1
echo ""
echo ""

# Test 6: Different JSON-RPC methods
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 6: Alternative JSON-RPC Methods"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "6a) eth_chainId"
curl -s -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
  "$SMART_ROUTER" | jq '.' 2>/dev/null || echo "Failed or invalid JSON"
echo ""

echo "6b) net_version"
curl -s -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_version","params":[],"id":1}' \
  "$SMART_ROUTER" | jq '.' 2>/dev/null || echo "Failed or invalid JSON"
echo ""

echo "6c) web3_clientVersion"
curl -s -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
  "$SMART_ROUTER" | jq '.' 2>/dev/null || echo "Failed or invalid JSON"
echo ""

# Test 7: Headers inspection
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 7: Response Headers Analysis"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -I -X POST \
  -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  "$SMART_ROUTER" 2>&1
echo ""

# Test 8: Try with different User-Agent
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 8: Request with Custom User-Agent"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  "$SMART_ROUTER" | jq '.' 2>/dev/null || echo "Failed or invalid JSON"
echo ""

# Test 9: Response timing
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 9: Response Timing (10 consecutive requests)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for i in {1..10}; do
  TIMING=$(curl -w "Time: %{time_total}s | HTTP: %{http_code} | Size: %{size_download} bytes\n" \
    -s -o /dev/null \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "$SMART_ROUTER")
  echo "Request $i: $TIMING"
  sleep 0.5
done
echo ""

# Test 10: Check for rate limiting
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 10: Rate Limiting Test (20 rapid requests)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
SUCCESS=0
FAILED=0
for i in {1..20}; do
  HTTP_CODE=$(curl -w "%{http_code}" -s -o /dev/null \
    -X POST \
    -H "Content-Type: application/json" \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "$SMART_ROUTER")
  
  if [ "$HTTP_CODE" = "200" ]; then
    ((SUCCESS++))
    echo -n "✓"
  else
    ((FAILED++))
    echo -n "✗($HTTP_CODE)"
  fi
done
echo ""
echo "Results: $SUCCESS success, $FAILED failed"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "DIAGNOSTIC COMPLETE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Summary:"
echo "  - If DNS failed: Check network connectivity"
echo "  - If TCP failed: Firewall or routing issue"
echo "  - If SSL failed: Certificate issue"
echo "  - If HTTP 200 but empty body: Smart Router internal issue"
echo "  - If rate limiting detected: Need to reduce request frequency"
echo ""

