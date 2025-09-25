#!/bin/bash

# Simple Commission Test Script
# Quick test to verify default commission value

set -e

# Configuration
CHAIN_ID="ETH1"
STAKE_AMOUNT="500000000000ulava"
WALLET_NAME="servicer1"
EXPECTED_COMMISSION="75"

echo "🔍 Testing Default Commission Value"
echo "================================="

# 1. Check if node is running
echo "1. Checking node status..."
if ! lavad status &> /dev/null; then
    echo "❌ Node is not running"
    exit 1
fi
echo "✅ Node is running"

# 2. Check wallet
echo "2. Checking wallet..."
if ! lavad keys show "$WALLET_NAME" &> /dev/null; then
    echo "❌ Wallet '$WALLET_NAME' not found"
    exit 1
fi
WALLET_ADDRESS=$(lavad keys show "$WALLET_NAME" -a)
echo "✅ Wallet found: $WALLET_ADDRESS"

# 3. Check if provider exists
echo "3. Checking existing provider..."
if lavad query epochstorage provider-metadata "$WALLET_ADDRESS" &> /dev/null; then
    echo "✅ Provider exists, checking commission..."
    
    # Get commission value
    COMMISSION=$(lavad query epochstorage provider-metadata "$WALLET_ADDRESS" --output json 2>/dev/null | jq -r '.MetaData[0].delegate_commission' 2>/dev/null || echo "")
    
    if [ "$COMMISSION" = "$EXPECTED_COMMISSION" ]; then
        echo "✅ SUCCESS: Commission is $COMMISSION (expected $EXPECTED_COMMISSION)"
        echo "🎉 Default commission test PASSED"
    else
        echo "❌ FAILED: Commission is $COMMISSION (expected $EXPECTED_COMMISSION)"
        echo "💡 Try staking a new provider with: ./test_commission_default.sh"
        exit 1
    fi
else
    echo "⚠️  No existing provider found"
    echo "💡 Run the full test script to stake a provider: ./test_commission_default.sh"
    
    # Show help for manual staking
    echo ""
    echo "Or stake manually with:"
    echo "lavad tx pairing stake-provider \"$CHAIN_ID\" \"$STAKE_AMOUNT\" \"127.0.0.1:2221,1\" 1 \$(lavad query staking validators --output json | jq -r '.validators[0].operator_address') -y --from \"$WALLET_NAME\" --provider-moniker \"test-provider\" --gas-adjustment \"1.5\" --gas \"auto\" --gas-prices \"0.00002ulava\""
fi

echo "================================="
