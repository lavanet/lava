#!/bin/bash
# Simple Error Response Logging Test
# Logs all curl outputs to error_responses_log.txt for manual review

LOG_FILE="error_responses_log.txt"
REST_ENDPOINT="http://127.0.0.1:3360"
TENDERMINT_ENDPOINT="http://127.0.0.1:3361"
CACHE_HEADER="lava-force-cache-refresh: true"

# Start fresh log
echo "=== Lava Error Response Test Log ===" > "$LOG_FILE"
echo "Test Date: $(date)" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

log_test() {
    local test_name="$1"
    local curl_cmd="$2"
    
    echo "======================================" >> "$LOG_FILE"
    echo "TEST: $test_name" >> "$LOG_FILE"
    echo "Time: $(date '+%H:%M:%S')" >> "$LOG_FILE"
    echo "Command: $curl_cmd" >> "$LOG_FILE"
    echo "--------------------------------------" >> "$LOG_FILE"
    echo "Response:" >> "$LOG_FILE"
    
    # Execute curl and log output
    eval "$curl_cmd -H '${CACHE_HEADER}'" 2>&1 | tee -a "$LOG_FILE" | jq . 2>/dev/null || eval "$curl_cmd -H '${CACHE_HEADER}'" 2>&1 >> "$LOG_FILE"
    
    echo "" >> "$LOG_FILE"
    echo "" >> "$LOG_FILE"
}

echo "Starting tests... Output will be logged to $LOG_FILE"
echo ""

# Test 1: REST Method Not Found (Code 12 - no structured extraction)
log_test \
    "REST - Method Not Found (Code 12)" \
    "curl -s -X GET \"${REST_ENDPOINT}/invalid_path\""

# Test 2: REST with invalid address (typically returns Code 3 - node validation error, no extraction)
log_test \
    "REST - Node Validation Error (Code 3, node pass-through)" \
    "curl -s -X GET \"${REST_ENDPOINT}/cosmos/bank/v1beta1/balances/invalid_address\""

# Test 3: REST Success (baseline)
log_test \
    "REST - Success Response (baseline)" \
    "curl -s -X GET \"${REST_ENDPOINT}/cosmos/base/tendermint/v1beta1/blocks/latest\""

# Test 4: TendermintRPC GET (may return -32603 aggregated error - no structured extraction)
log_test \
    "TendermintRPC GET - Aggregated Error (Code -32603, no error code extraction expected)" \
    "curl -s -X GET \"${TENDERMINT_ENDPOINT}/invalid_method\""

# Test 5: TendermintRPC POST Invalid Method (Code -32601 - no structured extraction)
log_test \
    "TendermintRPC POST - Method Not Found with ID 42" \
    "curl -s -X POST \"${TENDERMINT_ENDPOINT}\" -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"method\":\"invalid_method\",\"params\":[],\"id\":42}'"

# Test 6: TendermintRPC POST Invalid Method with String ID
log_test \
    "TendermintRPC POST - Method Not Found with String ID" \
    "curl -s -X POST \"${TENDERMINT_ENDPOINT}\" -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"method\":\"bad_method\",\"params\":[],\"id\":\"test-request-123\"}'"

# Test 7: TendermintRPC Success (baseline)
log_test \
    "TendermintRPC POST - Success Response (baseline)" \
    "curl -s -X POST \"${TENDERMINT_ENDPOINT}\" -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"method\":\"status\",\"params\":[],\"id\":1}'"

# Test 8: REST Node Error Pass-through (Code 3)
log_test \
    "REST - Node Error Pass-through (Code 3)" \
    "curl -s -X GET \"${REST_ENDPOINT}/cosmos/base/tendermint/v1beta1/blocks/invalid_height\""

echo ""
echo "======================================" >> "$LOG_FILE"
echo "VALIDATION CHECKLIST" >> "$LOG_FILE"
echo "======================================" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"
echo "Review the log and check:" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"
echo "[ ] REST Method Not Found (Test 1):" >> "$LOG_FILE"
echo "    - Has code: 12" >> "$LOG_FILE"
echo "    - Has empty details: []" >> "$LOG_FILE"
echo "    - NO error_code in details (expected)" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"
echo "[ ] REST Node Validation Error (Test 2):" >> "$LOG_FILE"
echo "    - Has code: 3" >> "$LOG_FILE"
echo "    - Has empty details: []" >> "$LOG_FILE"
echo "    - Message is from blockchain node (pass-through)" >> "$LOG_FILE"
echo "    - NO error_code extraction (node error, not Lava error)" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"
echo "[ ] TendermintRPC Method Not Found (Tests 5,6):" >> "$LOG_FILE"
echo "    - Has error.code: -32601" >> "$LOG_FILE"
echo "    - Has error.data with error details string" >> "$LOG_FILE"
echo "    - NO |Code: in data field (method not found, no relay)" >> "$LOG_FILE"
echo "    - ID is preserved from request" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"
echo "[ ] TendermintRPC Aggregated Error (Test 4):" >> "$LOG_FILE"
echo "    - Has error.code: -32603" >> "$LOG_FILE"
echo "    - Has error.data: 'GUID:xxx' format (no |Code: suffix)" >> "$LOG_FILE"
echo "    - ID is preserved (-1 for GET requests)" >> "$LOG_FILE"
echo "    - Note: Aggregated errors do NOT have |Code: extraction (known limitation)" >> "$LOG_FILE"
echo "" >> "$LOG_FILE"

echo "======================================" >> "$LOG_FILE"
echo "Test completed at: $(date)" >> "$LOG_FILE"
echo "======================================" >> "$LOG_FILE"

echo "✅ All tests complete!"
echo ""
echo "📄 Output logged to: $LOG_FILE"
echo ""
echo "To review the log:"
echo "  cat $LOG_FILE"
echo ""
echo "To search for specific errors:"
echo "  grep -A 20 'error_code' $LOG_FILE"
echo "  grep -A 20 'Code 12' $LOG_FILE"
echo "  grep -A 20 'Code 3' $LOG_FILE"
echo ""
echo "Note: Structured error extraction (|Code:XXXX) is NOT expected for:"
echo "  - Aggregated 'failed relay, insufficient results' errors (known limitation)"
echo "  - These errors go through mergeAllErrors() which converts to plain strings"
