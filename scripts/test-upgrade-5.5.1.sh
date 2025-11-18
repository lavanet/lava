#!/bin/bash
# Script to test v5.5.1 upgrade locally
# Usage: ./scripts/test-upgrade-5.5.1.sh <upgrade_height>

set -e

UPGRADE_VERSION="v5.5.1"
UPGRADE_HEIGHT=${1:-200}  # Default to block 200 if not specified

echo "================================================"
echo "Testing upgrade to $UPGRADE_VERSION"
echo "Upgrade height: $UPGRADE_HEIGHT"
echo "================================================"

# Step 1: Submit upgrade proposal
echo ""
echo "Step 1: Submitting upgrade proposal..."
lavad tx gov submit-legacy-proposal software-upgrade $UPGRADE_VERSION \
  --title "Upgrade to $UPGRADE_VERSION" \
  --upgrade-height $UPGRADE_HEIGHT \
  --no-validate \
  --gas "auto" \
  --from alice \
  --description "Testing upgrade to $UPGRADE_VERSION locally" \
  --keyring-backend "test" \
  --gas-prices "0.000000001ulava" \
  --gas-adjustment "1.5" \
  --deposit 10000000ulava \
  --chain-id lava \
  --node tcp://localhost:26657 \
  --yes

echo ""
echo "Waiting 3 seconds for proposal to be created..."
sleep 3

# Step 2: Vote on the proposal
echo ""
echo "Step 2: Voting YES on the proposal..."
# Get the latest proposal ID (should be the one we just created)
PROPOSAL_ID=$(lavad query gov proposals --output json --node tcp://localhost:26657 | jq '.proposals | length')

echo "Proposal ID: $PROPOSAL_ID"

lavad tx gov vote $PROPOSAL_ID yes -y \
  --from alice \
  --gas-adjustment "1.5" \
  --gas "auto" \
  --gas-prices "0.000000001ulava" \
  --keyring-backend "test" \
  --chain-id lava \
  --node tcp://localhost:26657

echo ""
echo "================================================"
echo "Upgrade proposal submitted and voted!"
echo "================================================"
echo ""
echo "The chain will halt at block $UPGRADE_HEIGHT"
echo ""
echo "Next steps:"
echo "1. Wait for the chain to reach block $UPGRADE_HEIGHT and halt"
echo "2. Build the new binary with: make build-lavad"
echo "3. Restart the chain with: lavad start"
echo "4. The upgrade handler will run automatically"
echo "5. Restart any providers/consumers you have running"
echo "================================================"

