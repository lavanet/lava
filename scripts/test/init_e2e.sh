#!/bin/bash 
killall lavap
set -e

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh

GASPRICE="0.00002ulava"

# Specs proposal
echo ---- Specs proposal ----
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/ibc.json,./cookbook/specs/tendermint.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/lava.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6 # need to sleep because plan policies need the specs when setting chain policies verifications

# Plans proposal
echo ---- Plans proposal ----
wait_next_block
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/emergency-mode.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 6

# Plan removal (of one)
echo ---- Plans removal ----
wait_next_block
# delete plan that deletes "temporary add" plan
lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/test_plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_next_block
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

STAKE="500000000000ulava"
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2221,1,archive,debug" 1 $(operator_address) -y --from servicer1  --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2222,1" 1 $(operator_address) -y --from servicer2  --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2223,1" 1 $(operator_address) -y --from servicer3  --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2224,1" 1 $(operator_address) -y --from servicer4  --provider-moniker "servicer4" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2225,1" 1 $(operator_address) -y --from servicer5  --provider-moniker "servicer5" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava tendermint/rest providers
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2261,1" 1 $(operator_address) -y --from servicer6  --provider-moniker "servicer6" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2262,1" 1 $(operator_address) -y --from servicer7  --provider-moniker "servicer7" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2263,1" 1 $(operator_address) -y --from servicer8  --provider-moniker "servicer8" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2264,1" 1 $(operator_address) -y --from servicer9  --provider-moniker "servicer9" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2265,1" 1 $(operator_address) -y --from servicer10  --provider-moniker "servicer10"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# subscribed clients
# (actually, only user1 is used in testutils/e2e/e2e.go, but having same count
# in both chains simplifies the e2e logic that expects same amount of staked
# clients in all tested chains)
lavad tx subscription buy "EmergencyModePlan" -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "DefaultPlan" -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "DefaultPlan" -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx subscription buy "EmergencyModePlan" -y --from user5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Test plan upgrade
echo ---- Subscription plan upgrade ----
wait_next_block
# test we have the plan active.
plan_index=$(lavad q subscription current $(lavad keys show user1 -a) | yq .sub.plan_index)
if [ "$plan_index" != "EmergencyModePlan" ]; then "echo subscription ${user1addr}: wrong plan index $plan_index .sub.plan_index doesn't contain EmergencyModePlan"; exit 1; fi
# buy the upgraded subscription
lavad tx subscription buy "DefaultPlan" -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# wait for the new subscription to take effect (1 epoch + 1 block as changes happen to subscription module after epochstorage module on the begin block events)
wait_next_block # wait block is here in case 1 block before epoch change we commit but the effect happens only on the epoch change meaning we didn't really wait an epoch for the changes to take effect.
sleep_until_next_epoch
# validate the new subscription is the default plan and not emergency mode plan.
plan_index=$(lavad q subscription current $(lavad keys show user1 -a) | yq .sub.plan_index)
if [ "$plan_index" != "DefaultPlan" ]; then "echo subscription ${user1addr}: wrong plan index $plan_index .sub.plan_index doesn't contain DefaultPlan"; exit 1; fi

user3addr=$(lavad keys show user3 -a)
# add debug addons and archive 
wait_next_block
lavad tx project set-policy $(lavad keys show user1 -a)-admin ./testutil/e2e/e2eConfigs/policies/consumer_policy.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
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


# validate deleted plan is removed. 
# Fetch the plans list
plans_list=$(lavad q plan list | yq .plans_info)

# Check if "to_delete_plan" exists
if echo "$plans_list" | grep -q '"index": "to_delete_plan"'; then
    echo "Index 'to_delete_plan' exists."
    exit 1 # fail test.
else
    echo "Index 'to_delete_plan' was removed successfully validation passed."
fi



# the end

