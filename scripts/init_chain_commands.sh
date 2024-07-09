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
ENVOY_LISTENER="envoy:9901"

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
CHAINS="ETH1,SEP1,HOL1,OSMOSIS,FTM250,CELO,LAV1,OSMOSIST,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COSMOSHUB,POLYGON1,EVMOS,OPTM,BASES,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM,NEAR,SQDSUBGRAPH,AGR,AGRT,KOIIT,AVAXT,CELESTIATM"
BASE_CHAINS="LAV1"
# stake providers on all chains
echo; echo "#### Staking provider 1 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$ENVOY_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 2 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$ENVOY_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer2 --provider-moniker "servicer2" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo; echo "#### Staking provider 3 ####"
lavad tx pairing bulk-stake-provider $BASE_CHAINS $PROVIDERSTAKE "$ENVOY_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer3 --provider-moniker "servicer3" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


echo; echo "#### Waiting 1 epoch ####"
sleep_until_next_epoch
