# Function to show ticker (full cycle)
display_ticker() {
  spinner="-\\|/-\\|/"
  for i in $(seq 0 ${#spinner}); do
      echo -ne "\r${spinner:i:1}"
      sleep 0.1
  done
}

# Function to wait until next epoch
sleep_until_next_epoch() {
  epoch_start=$(lavad query epochstorage show-epoch-details | grep "startBlock: ")
  echo "Waiting for the next epoch for the changes to be active"
  while true; do
    epoch_now=$(lavad query epochstorage show-epoch-details | grep "startBlock: ")
    display_ticker
    if [ "$epoch_start" != "$epoch_now" ]; then
      echo "finished waiting"
      break
    fi
  done
}

# Function to wait until next block
wait_next_block() {
  local max_attempts=30  # 30 seconds max
  local attempt=0
  
  # Get current block height (with retry logic)
  while [ $attempt -lt 10 ]; do
    current=$(lavad q block 2>/dev/null | jq -r '.block.header.height // empty' 2>/dev/null)
    if [ -n "$current" ] && [ "$current" != "null" ]; then
      break
    fi
    attempt=$((attempt + 1))
    sleep 1
  done
  
  if [ -z "$current" ] || [ "$current" == "null" ]; then
    echo "ERROR: Cannot query blockchain after 10 seconds"
    return 1
  fi
  
  echo "waiting for next block $current"
  attempt=0
  
  while true; do
    display_ticker
    new=$(lavad q block 2>/dev/null | jq -r '.block.header.height // empty' 2>/dev/null)
    if [ -n "$new" ] && [ "$new" != "null" ] && [ "$current" != "$new" ]; then
      echo "finished waiting at block $new"
      break
    fi
    attempt=$((attempt + 1))
    if [ $attempt -ge $max_attempts ]; then
      echo "ERROR: Timeout waiting for next block after $max_attempts seconds (stuck at $current)"
      return 1
    fi
    sleep 1
  done
}

# Wait for transaction to be included and successful
wait_for_tx() {
  local tx_hash=$1
  local max_attempts=${2:-30}
  local attempt=0
  
  echo "Waiting for transaction $tx_hash to be included..."
  
  while [ $attempt -lt $max_attempts ]; do
    # Query transaction and check if it exists and succeeded
    local tx_result=$(lavad q tx $tx_hash --output json 2>/dev/null)
    local tx_code=$(echo "$tx_result" | jq -r '.code // "null"' 2>/dev/null)
    
    if [ "$tx_code" = "0" ]; then
      echo "Transaction $tx_hash succeeded"
      return 0
    elif [ "$tx_code" != "null" ] && [ "$tx_code" != "" ]; then
      echo "ERROR: Transaction $tx_hash failed with code $tx_code"
      local raw_log=$(echo "$tx_result" | jq -r '.raw_log // ""' 2>/dev/null)
      echo "Error details: $raw_log"
      return 1
    fi
    
    attempt=$((attempt + 1))
    sleep 1
  done
  
  echo "ERROR: Timeout waiting for transaction $tx_hash after $max_attempts seconds"
  return 1
}

# Extract transaction hash from lavad tx command output
extract_tx_hash() {
  local output="$1"
  echo "$output" | grep -o 'txhash: [A-F0-9]*' | cut -d' ' -f2
}

# Wait for next block AND ensure a transaction is included (if tx_hash provided)
wait_next_block_and_tx() {
  local tx_output="$1"
  
  # First wait for next block
  if ! wait_next_block; then
    return 1
  fi
  
  # If transaction output was provided, verify it was included
  if [ -n "$tx_output" ]; then
    local tx_hash=$(extract_tx_hash "$tx_output")
    if [ -n "$tx_hash" ]; then
      if ! wait_for_tx "$tx_hash"; then
        return 1
      fi
    fi
  fi
  
  return 0
}

current_block() {
  current=$(lavad q block 2>/dev/null | jq -r '.block.header.height // empty' 2>/dev/null)
  if [ -z "$current" ] || [ "$current" == "null" ]; then
    echo "0"
  else
    echo $current
  fi
}

# function to wait until count ($1) blocks complete
wait_count_blocks() {
  for i in $(seq 1 $1); do
    wait_next_block
  done
}

# Function to check if a command is available
command_exists() {
  command -v "$1" >/dev/null 2>&1
}

# Function to check the Go version is at least 1.21
check_go_version() {
  local GO_VERSION=1.21

  if ! command_exists go; then
    return 1
  fi

  go_version=$(go version)
  go_version_major_full=$(echo "$go_version" | awk '{print $3}')
  go_version_major_minor_patch=${go_version_major_full:2}
  IFS='.' read -r major minor patch <<< "$go_version_major_minor_patch"
  go_version_major="$major.$minor"

  result=$(echo "${go_version_major}-$GO_VERSION > 0" | bc -l)
  if [ "$result" -eq 1 ]; then
    return 0
  fi

  return 1
}

# Function to find the latest vote
latest_vote() {
  # Check if jq is not installed
  if ! command_exists jq; then
      echo "jq not found. Please install jq using the init_install.sh script or manually."
      exit 1
  fi
  lavad q gov proposals --output=json 2> /dev/null | jq '.proposals[].id' | wc -l
}

create_health_config() {
  local source_file="$1"
  local destination_file="${source_file%.*}_gen.yml"
  local subscription_address="$2"
  local provider_address_1="$3"
  local provider_address_2="$4"
  local provider_address_3="$5"

  # Use awk to find the line number of the comment
  local comment_line_number=$(awk '/#REPLACED/ {print NR; exit}' "$source_file")

  # If the comment is found, create a new file with the updated content
  if [ -n "$comment_line_number" ]; then
    # Use awk to update the content and create a new file
    awk -v line="$comment_line_number" -v sub_addr="$subscription_address" \
        -v prov_addr_1="$provider_address_1" -v prov_addr_2="$provider_address_2" \
        -v prov_addr_3="$provider_address_3" \
        '{
            if (NR <= line) {
                print;
            } else if (NR == line + 1) {
                print "subscription_addresses:";
                print "  - " sub_addr;
                print "provider_addresses:";
                print "  - " prov_addr_1;
                print "  - " prov_addr_2;
                print "  - " prov_addr_3;
            }
        }' "$source_file" > "$destination_file"

    echo "File $destination_file created successfully."
  else
    echo "Comment #REPLACED not found in the file."
    exit 1
  fi
}


# Function to extract latest_block_height from lavad status | jq
get_block_height() {
    lavad status 2>/dev/null | jq -r '.SyncInfo.latest_block_height'
}

wait_for_lava_node_to_start() {
    # Monitor changes in block height
    while true; do
        current_height=$(get_block_height)
        if [[ "$current_height" =~ ^[0-9]+$ && "$current_height" -gt 5 ]]; then
            echo "Block height is now $current_height which is larger than 5"
            break
        fi
        sleep 1  # Check every second
    done
}

operator_address() {
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        local result=$(lavad q staking validators -o json 2>/dev/null | jq -r '.validators[0].operator_address // empty' 2>/dev/null)
        
        if [ -n "$result" ] && [ "$result" != "null" ]; then
            echo "$result"
            return 0
        fi
        
        attempt=$((attempt + 1))
        sleep 1
    done
    
    echo "ERROR: Failed to get operator address after $max_attempts attempts" >&2
    return 1
}


validate_env() {
# Array of variables to check
required_vars=(
  ETH_RPC_WS SEP_RPC_WS HOL_RPC_WS FTM_RPC_HTTP CELO_HTTP
  CELO_ALFAJORES_HTTP ARB1_HTTP APTOS_REST STARKNET_RPC POLYGON_MAINNET_RPC
  OPTIMISM_RPC BASE_RPC BSC_RPC SOLANA_RPC SUI_RPC OSMO_REST OSMO_RPC OSMO_GRPC
  LAVA_REST LAVA_RPC LAVA_RPC_WS LAVA_GRPC GAIA_REST GAIA_RPC GAIA_GRPC JUNO_REST
  JUNO_RPC JUNO_GRPC EVMOS_RPC EVMOS_TENDERMINTRPC EVMOS_REST EVMOS_GRPC CANTO_RPC
  CANTO_TENDERMINT CANTO_REST CANTO_GRPC AXELAR_RPC_HTTP AXELAR_REST AXELAR_GRPC
  AVALANCH_PJRPC AVALANCHT_PJRPC FVM_JRPC NEAR_JRPC AGORIC_REST AGORIC_GRPC
  KOIITRPC AGORIC_RPC AGORIC_TEST_REST AGORIC_TEST_GRPC AGORIC_TEST_RPC
  STARGAZE_RPC_HTTP STARGAZE_REST STARGAZE_GRPC
)

echo ""
echo "---------------------------------------------"
echo "-              ENV Validation               -"
echo "---------------------------------------------"
# Check each variable and print a warning if not set
for var in "${required_vars[@]}"; do
  if [ -z "${!var}" ]; then
    echo "Warning: Variable '$var' is not set or is empty."
  fi
done
echo "---------------------------------------------"
echo "-            ENV Validation Done            -"
echo "---------------------------------------------"
}

get_base_specs() {
    local priority_specs=(
        "specs/mainnet-1/specs/ibc.json"
        "specs/mainnet-1/specs/cosmoswasm.json"
        "specs/mainnet-1/specs/tendermint.json"
        "specs/mainnet-1/specs/cosmossdk.json"
        "specs/testnet-2/specs/cosmossdkv45.json"
        "specs/testnet-2/specs/cosmossdk_full.json"
        "specs/mainnet-1/specs/cosmossdkv50.json"
        "specs/mainnet-1/specs/ethermint.json"
        "specs/mainnet-1/specs/ethereum.json"
        "specs/mainnet-1/specs/solana.json"
        "specs/mainnet-1/specs/aptos.json"
        "specs/mainnet-1/specs/btc.json"
    )

    (IFS=,; echo "${priority_specs[*]}")
}

get_hyperliquid_specs() {
    local priority_specs=(
        "specs/mainnet-1/specs/ibc.json"
        "specs/mainnet-1/specs/cosmoswasm.json"
        "specs/mainnet-1/specs/tendermint.json"
        "specs/mainnet-1/specs/cosmossdk.json"
        "specs/testnet-2/specs/cosmossdkv45.json"
        "specs/testnet-2/specs/cosmossdk_full.json"
        "specs/mainnet-1/specs/cosmossdkv50.json"
        "specs/testnet-2/specs/lava.json"
        "specs/mainnet-1/specs/hyperliquid.json"
    )

    (IFS=,; echo "${priority_specs[*]}")
}

get_all_specs() {
    # Ensure get_base_specs outputs elements correctly split into an array
    IFS=',' read -r -a priority_specs <<< "$(get_base_specs)" 

    local other_specs=()

    # Find all JSON spec files and store them in a temporary file
    find specs/{mainnet-1,testnet-2}/specs -name "*.json" > /tmp/specs_list.txt

    # Process each file from the find command
    while IFS= read -r file; do
        local is_priority=false
        local is_in_other_specs=false

        # Check if the file is in priority_specs
        for pspec in "${priority_specs[@]}"; do
            if [[ "$file" == "$pspec" ]]; then
                is_priority=true
                break
            fi
        done

        # Check if the file is already in other_specs
        if [[ "$is_priority" == "false" ]]; then
            for ospec in "${other_specs[@]}"; do
                if [[ "$file" == "$ospec" ]]; then
                    is_in_other_specs=true
                    break
                fi
            done
        fi

        # Add the file to other_specs if it's not already there
        if [[ "$is_priority" == "false" && "$is_in_other_specs" == "false" ]]; then
            other_specs+=("$file")
        fi
    done < /tmp/specs_list.txt

    # Cleanup temporary file
    rm /tmp/specs_list.txt

    # Combine arrays and output as a comma-separated string
    (IFS=,; echo "${priority_specs[*]},${other_specs[*]}")
}
