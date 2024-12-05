#!/bin/bash
# call this bash with this format:
# test_spec_full [SPEC_file_path] {[api_interface] [service-url] ...} [--install]
# example:
# test_spec_full cookbook/specs/lava.json rest 127.0.0.1:1317 tendermintrpc 127.0.0.1:26657 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

if [[ "${@: -1}" == "--install" ]]; then
    end=$(( $# - 1 ))  # Exclude --install from processing
    # If --install flag is found, perform the installation
    echo "Install flag detected. Performing installation..."
    ${__dir}/init_install.sh
else
    end=$#
fi

dry=false
if [[ "${@: -1}" == "--dry" ]]; then
    end=$(( $# - 1 ))  # Exclude --dry from processing
    dry=true
else
    end=$#
fi

interfaces=()
urls=()
# Loop through the arguments
for ((i = 2; i <= end; i+=2))
do
    interfaces+=("${!i}")
done

for ((i = 3; i <= end; i+=2))
do
    urls+=("${!i}")
done

LOGS_DIR=${__dir}/../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log
rm $LOGS_DIR/*.yml

proposal_indexes=$(jq -r '.proposal.specs[].index' "$1")
index_list=()

while IFS= read -r line; do
    index_list+=("$line")
done <<< "$proposal_indexes"

echo "processing indexes: ${index_list[@]}"
killall screen
screen -wipe

## Handle Node ##

if [ "$dry" = false ]; then
    echo "[Test Setup] installing all binaries"
    make install-all 

    echo "[Test Setup] setting up a new lava node"
    screen -d -m -S node bash -c "${__dir}/start_env_dev.sh"
    screen -ls
    echo "[Test Setup] sleeping 5 seconds for node to finish setup (if its not enough increase timeout)"
    sleep 5
    wait_for_lava_node_to_start

    GASPRICE="0.00002ulava"
    # add all existing specs so inheritance works
    lavad tx gov submit-legacy-proposal spec-add ${__dir}/../cookbook/specs/ibc.json,${__dir}/../cookbook/specs/cosmoswasm.json,${__dir}/../cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,${__dir}/../cookbook/specs/cosmossdk_45.json,${__dir}/../cookbook/specs/cosmossdk_full.json,${__dir}/../cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,${__dir}/../cookbook/specs/cosmoshub.json,${__dir}/../cookbook/specs/lava.json,${__dir}/../cookbook/specs/osmosis.json,${__dir}/../cookbook/specs/fantom.json,${__dir}/../cookbook/specs/celo.json,${__dir}/../cookbook/specs/optimism.json,${__dir}/../cookbook/specs/arbitrum.json,${__dir}/../cookbook/specs/starknet.json,${__dir}/../cookbook/specs/aptos.json,${__dir}/../cookbook/specs/juno.json,${__dir}/../cookbook/specs/polygon.json,${__dir}/../cookbook/specs/evmos.json,${__dir}/../cookbook/specs/base.json,${__dir}/../cookbook/specs/canto.json,${__dir}/../cookbook/specs/sui.json,${__dir}/../cookbook/specs/solana.json,${__dir}/../cookbook/specs/bsc.json,${__dir}/../cookbook/specs/axelar.json,${__dir}/../cookbook/specs/avalanche.json,${__dir}/../cookbook/specs/fvm.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
    wait_next_block
    wait_next_block
    lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    sleep 4

    # Plans proposal
    lavad tx gov submit-legacy-proposal plans-add ${__dir}/../cookbook/plans/test_plans/default.json,${__dir}/../cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    wait_next_block
    wait_next_block
    lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    wait_next_block
    wait_next_block
    lavad tx gov submit-legacy-proposal spec-add "$1" --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
    wait_next_block
    wait_next_block
    lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    sleep 4

    PROVIDERSTAKE="500000000000ulava"

    # do not change this port without modifying the input_yaml
    PROVIDER1_LISTENER="127.0.0.1:2220"

    lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
    echo "[+] processing the next indexes:${index_list[@]}"
    for index in ${index_list[@]}
    do
        echo "Processing index: $index"
        lavad tx pairing stake-provider "$index" $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --from servicer1 --provider-moniker "provider-$index" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
        wait_next_block
    done

    sleep_until_next_epoch
fi
valid_interfaces=("rest" "jsonrpc" "grpc" "tendermintrpc")

for interface in "$interfaces"
do
    # Check if the argument is a valid interface
    if [[ ! " ${valid_interfaces[@]} " =~ " $interface " ]]; then
        echo "invalid api interface. ${interface}"
        exit 1 
    fi
done

if [ ${#interfaces[@]} -ne ${#urls[@]} ]; then
    echo "Arrays have different lengths. ${interfaces[@]} vs ${urls[@]} vs ${args[@]}"
    exit 1 
fi

## Handle Provider ##

input_yaml="${__dir}/../config/provider_examples/test_spec_template.yml" #if testing archive change to test_spec_template_archive.yml
output_yaml="${LOGS_DIR}/provider.yml"

line_numbers=()
while IFS= read -r line; do
    line_numbers+=("$line")
done < <(awk '/- api-interface:/ { print NR }' "$input_yaml")

copy_content() {
    local start_line="$1"
    local end_line="$2"
    local index="$3"
    local url="$4"
    awk -v s="$start_line" -v e="$end_line" -v idx="$index" -v url="$url" 'NR >= s && NR <= e {sub("REPLACE_THIS_URL", url, $0);sub("INDEX_REPLACE_THIS", idx, $0);print}' "$input_yaml" >> "$output_yaml"
}

head -n 1 "$input_yaml" >> "$output_yaml"
# Copy blocks to provider.yaml based on line numbers and args
handled_indices=()
for ((j = 0; j < ${#index_list[@]}; j++))
do
    for ((i = 0; i < ${#line_numbers[@]}; i++))
    do
        start_line=${line_numbers[$i]}
        interface=$(awk -v ln="$start_line" 'NR == ln {print $3}' "$input_yaml" | tr -d '"')
        for interface_idx in "${!interfaces[@]}"; do
            if [[ "${interfaces[$interface_idx]}" == "$interface" ]] && ! [[ " ${handled_indices[@]} " =~ " $interface_idx " ]]; then
                if ((i == ${#line_numbers[@]} - 1)); then
                    # If it's the last interface, copy till the end of the file
                    copy_content "$start_line" "$(wc -l < "$input_yaml")" "${index_list[$j]}" "${urls[$interface_idx]}"
                else
                    end_line=${line_numbers[$((i + 1))]}
                    ((end_line--)) # Decrement end line to avoid copying the next interface block
                    copy_content "$start_line" "$end_line" "${index_list[$j]}" "${urls[$interface_idx]}"
                fi
                handled_indices+=("$interface_idx")
                break  # Stop loop when found
            fi
        done
    done
done
echo "[+]generated provider config: $output_yaml"
cat $output_yaml

echo "[Test Setup] starting provider"
if [ "$dry" = false ]; then
    screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider testutil/debugging/logs/provider.yml $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --chain-id lava --metrics-listen-address ":7776" 2>&1 | tee $LOGS_DIR/PROVIDER1.log"
fi
## Handle Consumer ##

port=3000
template="\
    - chain-id: CHAIN_ID_REPLACE
      api-interface: API_INTERFACE_REPLACE
      network-address: 127.0.0.1:PORT_REPLACE
"
output_consumer_yaml="${LOGS_DIR}/consumer.yml"
echo "endpoints:" >> $output_consumer_yaml
test_consumer_command_args=""
for ((j = 0; j < ${#index_list[@]}; j++))
do
    for ((i = 0; i < ${#line_numbers[@]}; i++))
    do
        start_line=${line_numbers[$i]}
        interface=$(awk -v ln="$start_line" 'NR == ln {print $3}' "$input_yaml" | tr -d '"')
        for interface_idx in "${!interfaces[@]}"; do
            if [[ "${interfaces[$interface_idx]}" == "$interface" ]]; then
                port=$((port + 1))
                replaced=$(echo "$template" | sed -e "s/CHAIN_ID_REPLACE/${index_list[$j]}/" -e "s/API_INTERFACE_REPLACE/${interface}/" -e "s/PORT_REPLACE/${port}/")
                echo "$replaced" >> $output_consumer_yaml
                if [ "$interface" != "grpc" ]; then
                    url_test="http://127.0.0.1:$port"
                else
                    url_test="127.0.0.1:$port"
                fi
                test_consumer_command_args+=" $url_test ${index_list[$j]} $interface"
                break
            fi
        done
    done
done

echo "[+]generated consumer config: $output_consumer_yaml"
cat $output_consumer_yaml
if [ "$dry" = false ]; then
    screen -d -m -S consumers bash -c "source ~/.bashrc; lavap rpcconsumer testutil/debugging/logs/consumer.yml $EXTRA_PORTAL_FLAGS --geolocation 1 --debug-relays --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --metrics-listen-address ":7779" 2>&1 | tee $LOGS_DIR/CONSUMER.log"

    echo "[+] letting providers start and running health check then running command with flags: $test_consumer_command_args"
    sleep 10
    lavap test rpcconsumer$test_consumer_command_args --chain-id=lava
fi
