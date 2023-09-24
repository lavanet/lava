#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.000000001ulava"

# ,./cookbook/specs/spec_add_mantle.json
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_45.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json,./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json,./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json,./cookbook/specs/spec_add_avalanche.json,./cookbook/specs/spec_add_fvm.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_count_blocks 2
echo "submitted first proposal"
echo "latest vote2: $(latest_vote)"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_count_blocks 4
echo "voted on first proposal"
sleep 4
lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/default.json,./cookbook/plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 2
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 4

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"

sleep 4

lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 2
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx  subscription buy DefaultPlan $(lavad keys show user1 -a) -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# lavad tx project set-policy $(lavad keys show user1 -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# MANTLE
CHAINS="ETH1,GTH1,COS3,FTM250,CELO,LAV1,COS4,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COS5,POLYGON1,EVMOS,OPTM,BASET,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM"
# stake providers on all chains
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER2_LISTENER,1" 1 -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER3_LISTENER,1" 1 -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch

lavad tx dualstaking delegate $(lavad keys show servicer1 -a) ETH1 $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer2 -a) ETH1 $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 1
lavad tx dualstaking delegate $(lavad keys show servicer3 -a) ETH1 $PROVIDERSTAKE -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

. ${__dir}/setup_providers.sh
