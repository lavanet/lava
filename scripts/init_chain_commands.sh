#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.000000001ulava"

echo; echo "#### Sending proposal for specs ####"
# ,./cookbook/specs/spec_add_mantle.json
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_45.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json,./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json,./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json,./cookbook/specs/spec_add_avalanche.json,./cookbook/specs/spec_add_fvm.json,./cookbook/specs/spec_add_near.json,./cookbook/specs/spec_add_sqdsubgraph.json,./cookbook/specs/spec_add_agoric.json,./cookbook/specs/spec_add_koii.json,./cookbook/specs/spec_add_stargaze.json,./cookbook/specs/spec_add_blast.json,./cookbook/specs/spec_add_secret.json --lava-dev-test -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
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

sleep 4

echo; echo "#### Sending proposal for plans del ####"
lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/test_plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Waiting 2 blocks ####"
wait_count_blocks 2

echo; echo "#### Voting on plans del proposal ####"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Sending proposal for plans del ####"
lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) --enable-auto-renewal -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# wait_count_blocks 2
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# MANTLE
CHAINS="ETH1,GTH1,SEP1,OSMOSIS,FTM250,CELO,LAV1,OSMOSIST,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COSMOSHUB,POLYGON1,EVMOS,OPTM,BASET,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM,NEAR,SQDSUBGRAPH,AGR,AGRT,KOIIT,AVAXT"
BASE_CHAINS="ETH1,LAV1"
# stake providers on all chains
echo; echo "#### Staking provider 1 ####"
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 2 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 3 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

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
