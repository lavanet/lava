#!/bin/bash

# Quick Rate Limiting Test - Minimal validation script
# Use this for rapid testing of the rate limiting functionality

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${1}[$(date +'%H:%M:%S')] $2${NC}"
}

# Test configuration
PROVIDER_ADDR="lava@1f339hhulrspqy4e5s7tgc4zvkkq8vhr6q4c0ql"  # servicer1 from our testing
MOVE_AMOUNT="50000000000ulava"  # 50 LAVA

echo "🚀 Quick Rate Limiting Test"
echo "=========================="

# Check if environment is running
if ! lavad status >/dev/null 2>&1; then
    log $RED "❌ Lava chain not running. Please start the environment first:"
    echo "   ./scripts/pre_setups/init_lava_only_test.sh"
    exit 1
fi

# Check current block
CURRENT_BLOCK=$(lavad status | jq -r '.SyncInfo.latest_block_height')
log $YELLOW "📊 Current block: $CURRENT_BLOCK"

# Check provider stakes
echo ""
log $YELLOW "📋 Current provider stakes:"
lavad query pairing provider $PROVIDER_ADDR --output json | jq '.stakeEntries[] | {chain: .chain, stake: .stake.amount}'

# Check provider metadata (rate limit status)
echo ""
log $YELLOW "🔍 Provider metadata (rate limit info):"
METADATA=$(lavad query epochstorage provider-metadata $PROVIDER_ADDR)
echo "$METADATA" | grep -E "(last_stake_move|last_change)"

LAST_MOVE=$(echo "$METADATA" | grep "last_stake_move" | awk '{print $2}' | tr -d '"')
CURRENT_TIME=$(date +%s)
TIME_DIFF=$((CURRENT_TIME - LAST_MOVE))
HOURS_SINCE=$((TIME_DIFF / 3600))

echo ""
log $YELLOW "⏰ Time since last move: $HOURS_SINCE hours ($TIME_DIFF seconds)"

if [ $TIME_DIFF -ge 86400 ]; then
    log $GREEN "✅ Rate limit expired - moves should be allowed"
    EXPECTED="allowed"
else
    REMAINING_SECONDS=$((86400 - TIME_DIFF))
    REMAINING_HOURS=$((REMAINING_SECONDS / 3600))
    REMAINING_MINUTES=$(((REMAINING_SECONDS % 3600) / 60))
    log $RED "⏳ Rate limit active - $REMAINING_HOURS hours $REMAINING_MINUTES minutes remaining"
    EXPECTED="blocked"
fi

# Test stake move
echo ""
log $YELLOW "🧪 Testing stake move: LAV1 → ETH1 ($MOVE_AMOUNT)"

if lavad tx pairing move-provider-stake "LAV1" "ETH1" $MOVE_AMOUNT -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices "0.00002ulava" 2>/dev/null; then
    if [ "$EXPECTED" = "allowed" ]; then
        log $GREEN "✅ SUCCESS: Move allowed as expected (rate limit expired)"
    else
        log $RED "❌ UNEXPECTED: Move allowed but should be rate limited"
        exit 1
    fi
else
    if [ "$EXPECTED" = "blocked" ]; then
        log $GREEN "✅ SUCCESS: Move blocked as expected (rate limited)"
        # Show the error message
        echo ""
        log $YELLOW "📝 Rate limit error message:"
        lavad tx pairing move-provider-stake "LAV1" "ETH1" $MOVE_AMOUNT -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices "0.00002ulava" 2>&1 | grep -A2 "provider must wait" || true
    else
        log $RED "❌ UNEXPECTED: Move blocked but rate limit should be expired"
        exit 1
    fi
fi

# Show final states
echo ""
log $YELLOW "📊 Final provider stakes:"
lavad query pairing provider $PROVIDER_ADDR --output json | jq '.stakeEntries[] | {chain: .chain, stake: .stake.amount}'

echo ""
log $GREEN "🎉 Quick test completed!"

# Instructions for next steps
echo ""
echo "💡 Next steps:"
if [ "$EXPECTED" = "blocked" ]; then
    echo "   - Rate limiting is working correctly"
    echo "   - Wait $REMAINING_HOURS hours $REMAINING_MINUTES minutes for cooldown to expire"
    echo "   - Or test bulk operations: lavad tx pairing distribute-provider-stake \"LAV1,40,ETH1,35,SOLANA,25\" -y --from servicer1"
else
    echo "   - Rate limit has expired, normal operations resumed"
    echo "   - Test another move to start a new cooldown period"
    echo "   - Test bulk operations while rate limit is active"
fi
