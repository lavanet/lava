#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.00002ulava"

echo; echo "#### Sending proposal for specs ####"
cd ./cookbook/specs/
lavad tx gov submit-legacy-proposal spec-add ./ibc.json,./tendermint.json,./cosmoswasm.json,./cosmossdk.json,./cosmossdk_45.json,./cosmossdk_full.json,./ethermint.json,./ethereum.json,./cosmoshub.json,./lava.json,./osmosis.json,./fantom.json,./celo.json,./optimism.json,./arbitrum.json,./starknet.json,./aptos.json,./juno.json,./polygon.json,./evmos.json,./base.json,./canto.json,./sui.json,./solana.json,./bsc.json,./axelar.json,./avalanche.json,./fvm.json,./near.json,./sqdsubgraph.json,./agoric.json,./koii.json,./stargaze.json,./blast.json,./secret.json,./celestia.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
cd ../../
echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on specs proposal ####"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 4
sleep 4

echo; echo "#### Sending proposal for test plans add ####"
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans test add proposal ####"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 2

echo; echo "#### Sending proposal for plans add ####"
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/explorer.json,./cookbook/plans/adventurer.json,./cookbook/plans/whale.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans add proposal ####"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 4 blocks ####"
wait_count_blocks 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"
# PROVIDER4_LISTENER="127.0.0.1:2224"

sleep 4

echo; echo "#### Sending proposal for plans del ####"
lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/test_plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans del proposal ####"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Buy DefaultPlan subscription for user1 ####"
lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) --enable-auto-renewal -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# wait_count_blocks 2
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_extension.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# MANTLE
CHAINS="ETH1,SEP1,HOL1,OSMOSIS,FTM250,CELO,LAV1,OSMOSIST,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COSMOSHUB,POLYGON1,EVMOS,OPTM,BASES,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM,NEAR,SQDSUBGRAPH,AGR,AGRT,KOIIT,AVAXT,CELESTIATM"
BASE_CHAINS="ETH1,LAV1"
# stake providers on all chains
echo; echo "#### Staking provider 1 ####"
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50  --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 2 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50  --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 3 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50  --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# echo; echo "#### Staking provider 4 ####"
# lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER4_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50  --from servicer4 --provider-moniker "servicer4" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 1 ####"
lavad tx dualstaking delegate $(lavad keys show servicer1 -a) ETH1 $(operator_address) $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 2 ####"
lavad tx dualstaking delegate $(lavad keys show servicer2 -a) ETH1 $(operator_address) $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 1 block ####"
wait_count_blocks 1

echo; echo "#### Delegating provider 3 ####"
lavad tx dualstaking delegate $(lavad keys show servicer3 -a) ETH1 $(operator_address) $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
echo; echo "#### Waiting 1 epoch ####"
sleep_until_next_epoch

HEALTH_FILE="config/health_examples/health_template.yml"
create_health_config $HEALTH_FILE $(lavad keys show user1 -a) $(lavad keys show servicer2 -a) $(lavad keys show servicer3 -a)

lavad tx gov submit-legacy-proposal set-iprpc-data 1000000000ulava --min-cost 100ulava --add-subscriptions $(lavad keys show -a user1) --from alice -y
wait_count_blocks 1
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

if [[ "$1" != "--skip-providers" ]]; then
. ${__dir}/setup_providers.sh
echo "letting providers start and running health check"
sleep 10
lavap test health $HEALTH_FILE
fi
