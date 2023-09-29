#!/bin/bash 
killall lavap
set -e

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

GASPRICE="0.000000001ulava"

# Specs proposal
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_lava.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov deposit 1 100ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 6 # need to sleep because plan policies need the specs when setting chain policies verifications

# Plans proposal
wait_next_block
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/default.json,./cookbook/plans/emergency-mode.json,./cookbook/plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov deposit 2 100ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6

# Plan removal (of one)
wait_next_block
lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov deposit 3 1ulava -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

STAKE="500000000000ulava"
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2221,1" 1 -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2222,1" 1 -y --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2223,1" 1 -y --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2224,1" 1 -y --from servicer4 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2225,1" 1 -y --from servicer5 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava tendermint/rest providers
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2261,1" 1 -y --from servicer6 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2262,1" 1 -y --from servicer7 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2263,1" 1 -y --from servicer8 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2264,1" 1 -y --from servicer9 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2265,1" 1 -y --from servicer10 --provider-moniker "dummyMoniker"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# subscribed clients
# (actually, only user1 is used in testutils/e2e/e2e.go, but having same count
# in both chains simplifies the e2e logic that expects same amount of staked
# clients in all tested chains)
lavad tx subscription buy "DefaultPlan" -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "DefaultPlan" -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "DefaultPlan" -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "EmergencyModePlan" -y --from user5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

user3addr=$(lavad keys show user3 -a)

wait_next_block
lavad tx subscription add-project "myproject1" -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx subscription add-project "myproject" --policy-file ./cookbook/projects/example_policy.yml -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block

count=$(lavad q subscription list-projects ${user3addr} | grep "lava@" | wc -l)
if [ "$count" -ne 3 ]; then "echo subscription ${user3addr}: wrong project count $count instead of 3"; exit 1; fi

lavad tx project add-keys -y "$user3addr-myproject" --from user3 cookbook/projects/example_project_keys.yml --gas-prices=$GASPRICE
wait_next_block
lavad tx project del-keys -y "$user3addr-myproject" --from user3 cookbook/projects/example_project_keys.yml --gas-prices=$GASPRICE
sleep_until_next_epoch

lavad tx subscription del-project myproject -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep_until_next_epoch

count=$(lavad q subscription list-projects ${user3addr} | grep "lava@" | wc -l)
if [ "$count" -ne 2 ]; then "echo subscription ${user3addr}: wrong project count $count instead of 2"; exit 1; fi

# the end
