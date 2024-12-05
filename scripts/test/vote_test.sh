#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.00002ulava"

delegate_amount=1000000000000ulava
delegate_amount_big=49000000000000ulava
operator=$(lavad q staking validators --output json | jq -r ".validators[0].operator_address")
echo "operator: $operator"
lavad tx staking delegate $operator $delegate_amount --from bob --chain-id lava --gas-prices $GASPRICE --gas-adjustment 1.5 --gas auto -y
lavad tx staking delegate $operator $delegate_amount --from user1 --chain-id lava --gas-prices $GASPRICE --gas-adjustment 1.5 --gas auto -y
lavad tx staking delegate $operator $delegate_amount --from user2 --chain-id lava --gas-prices $GASPRICE --gas-adjustment 1.5 --gas auto -y
lavad tx staking delegate $operator $delegate_amount_big --from user3 --chain-id lava --gas-prices $GASPRICE --gas-adjustment 1.5 --gas auto -y
lavad tx staking delegate $operator $delegate_amount_big --from user4 --chain-id lava --gas-prices $GASPRICE --gas-adjustment 1.5 --gas auto -y
wait_count_blocks 1
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2
# voting abstain with 50% of the voting power, yes with 2% of the voting power no with 1% of the voting power
lavad tx gov vote $(latest_vote) abstain -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote $(latest_vote) yes -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote $(latest_vote) yes -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote $(latest_vote) no -y --from bob --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo "latest vote: $(latest_vote)"
lavad q gov proposal $(latest_vote)