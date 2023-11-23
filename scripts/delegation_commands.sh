#!/bin/bash

## This script assumes that you started a node and ran init_chain_commands.sh
## The state (relevant to delegations) that this script assumes:
##
## servicer1 is staked on all chains
## servicer2 and servicer3 are both staked only on ETH1 and LAV1
## user1 is delegated to all 3 providers on ETH1

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh

# Provider delegations (dualstaking module)
lavad tx dualstaking delegate $(lavad keys show servicer1 -a) COS3 1000ulava -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer3 -a) LAV1 1000ulava -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer2 -a) LAV1 100000ulava -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer1 -a) NEAR 100000ulava -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer1 -a) FTM250 100000ulava -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking redelegate $(lavad keys show servicer1 -a) ETH1 $(lavad keys show -a servicer2) LAV1 100000ulava -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking redelegate $(lavad keys show servicer1 -a) FTM250 $(lavad keys show -a servicer3) ETH1 1000ulava -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1
lavad tx dualstaking unbond $(lavad keys show servicer2 -a) ETH1 37000ulava -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava
wait_count_blocks 1

# Validator delegations (staking module)
val=$(lavad query staking validators | grep -oP 'operator_address: \K\S+' | head -n 1)   # gets the first validator address
lavad tx staking delegate $val 500ulava --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava -y
wait_count_blocks 1
lavad tx staking delegate $val 5000ulava --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava -y
wait_count_blocks 1
lavad tx staking delegate $val 50ulava --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava -y
wait_count_blocks 1
lavad tx staking unbond $val 50ulava --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava -y
wait_count_blocks 1
lavad tx staking unbond $val 500ulava --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava -y
wait_count_blocks 1