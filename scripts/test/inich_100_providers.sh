#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.000000001ulava"

echo; echo "#### Sending proposal for specs ####"
specs=$(get_all_specs)
lavad tx gov submit-legacy-proposal spec-add $specs --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on specs proposal ####"
lavad tx gov vote "$(latest_vote)" yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 4
sleep 4

echo; echo "#### Sending proposal for test plans add ####"
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans test add proposal ####"
lavad tx gov vote "$(latest_vote)" yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 2

echo; echo "#### Sending proposal for plans add ####"
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/explorer.json,./cookbook/plans/adventurer.json,./cookbook/plans/whale.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans add proposal ####"
lavad tx gov vote "$(latest_vote)" yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 4

PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"

sleep 4

echo; echo "#### Sending proposal for plans del ####"
lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/test_plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans del proposal ####"
lavad tx gov vote "$(latest_vote)" yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Sending proposal for plans del ####"
lavad tx subscription buy DefaultPlan "$(lavad keys show user1 -a)" --enable-auto-renewal -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# wait_count_blocks 2
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# MANTLE
CHAINS="ETH1,SEP1,HOL1,OSMOSIS,FTM250,CELO,LAV1,OSMOSIST,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COSMOSHUB,POLYGON1,EVMOS,OPTM,BASES,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM,NEAR,SQDSUBGRAPH,AGR,AGRT,KOIIT,AVAXT,CELESTIATM"
BASE_CHAINS="ETH1,LAV1"
# stake providers on all chains
echo; echo "#### Staking provider 1 ####"
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 "$(operator_address)" -y --delegate-commission 50  --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 2 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 "$(operator_address)" -y --delegate-commission 50  --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 3 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 "$(operator_address)" -y --delegate-commission 50  --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking 100 providers ####"
users=()
for i in $(seq 1 100); do
  users+=("useroren$i")
done

for user in "${users[@]}"; do
  lavad tx pairing stake-provider ETH1 600000000000ulava "127.0.0.1:2221,EU" EU "$(operator_address)" --from $user -y --provider-moniker $user --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava 
done

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 1 ####"
lavad tx dualstaking delegate "$(lavad keys show servicer1 -a)" ETH1 "$(operator_address)" $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 2 ####"
lavad tx dualstaking delegate "$(lavad keys show servicer2 -a)" ETH1 "$(operator_address)" $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 3 ####"
lavad tx dualstaking delegate "$(lavad keys show servicer3 -a)" ETH1 "$(operator_address)" $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
echo; echo "#### Waiting 1 epoch ####"
sleep_until_next_epoch

HEALTH_FILE="config/health_examples/health_template.yml"
create_health_config $HEALTH_FILE "$(lavad keys show user1 -a)" "$(lavad keys show servicer2 -a)" "$(lavad keys show servicer3 -a)"

lavad tx gov submit-legacy-proposal set-iprpc-data 1000000000ulava --min-cost 100ulava --add-subscriptions "$(lavad keys show -a user1)" --from alice -y
wait_count_blocks 1
lavad tx gov vote "$(latest_vote)" yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

for user in "${users[@]}"; do
  lavad tx pairing stake-provider ETH1 600000000000ulava "127.0.0.1:2221,EU" EU "$(operator_address)" --from $user -y --provider-moniker $user --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava 
done

wait_count_blocks 1

for user in "${users[@]}"; do
  lavad q pairing get-pairing ETH1 user1
done