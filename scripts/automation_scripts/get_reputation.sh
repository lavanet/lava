#!/bin/bash

# Check if chain ID argument is provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <chain_id>"
    exit 1
fi

CHAIN_ID=$1

# Get all provider addresses and their monikers
providers_output=$(lavad q pairing providers $CHAIN_ID)
if ! echo "$providers_output" | grep -q "address:"; then
    echo "$CHAIN_ID has no providers!"
    exit 0
fi

# Initialize an array for providers with no reputation
declare -a no_reputation=()

echo "$CHAIN_ID reputation:"
echo "Active providers:"

# Process each provider
while IFS= read -r line; do
    if [[ $line =~ address:\ *(lava@[a-z0-9]*) ]]; then
        addr="${BASH_REMATCH[1]}"
        # Get the moniker from the previous line
        moniker=$(echo "$providers_output" | grep -B 1 "$addr" | grep "moniker:" | awk '{print $2}')
        
        # Get reputation details
        reputation=$(lavad q pairing provider-reputation-details $addr $CHAIN_ID "*")
        score=$(echo "$reputation" | grep -A1 "reputation_pairing_score:" | tail -n 1 | awk '{print $2}' | tr -d '"')
        
        # Get rank and performance
        provider_info=$(lavad q pairing provider-reputation $addr $CHAIN_ID "*")
        rank=$(echo "$provider_info" | grep "rank:" | awk '{print $2}' | tr -d '"')
        performance=$(echo "$provider_info" | grep "overall_performance:" | cut -d ':' -f 2 | sed 's/^ *//')
        
        if [ ! -z "$score" ]; then
            printf "%s (%s): %s [Rank: %s, Performance: %s]\n" "$addr" "$moniker" "$score" "$rank" "$performance"
        else
            no_reputation+=("$addr ($moniker)")
        fi
    fi
done <<< "$providers_output"

# Print providers with no reputation if any exist
if [ ${#no_reputation[@]} -ne 0 ]; then
    echo -e "\nNo reputation:"
    printf '%s\n' "${no_reputation[@]}"
fi
