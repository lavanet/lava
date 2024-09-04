#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

trace() {
    (
        set -x
        "$@"
    )
}

txoptions="-y --from alice --gas-adjustment 1.5 --gas auto --gas-prices 0.00002ulava"

# run the chain
echo "setting up lava chain"

./scripts/start_env_dev.sh > chainlogs.txt 2>&1 &


current="0"
count=1
while [[ $current != "5" ]]
do
    ((count++))
    if ((count > 300)); then
        echo "timeout: Failed to start the chain"
        cat chainlogs.txt
        killall lavad
        exit 1 
    fi
    sleep 1
    current=$(current_block 2>/dev/null) || true
done
echo "lava chain is running"
set -e

echo "Testing epochstorage q commands"
trace lavad q epochstorage list-fixated-params >/dev/null
trace lavad q epochstorage list-stake-storage >/dev/null
trace lavad q epochstorage params >/dev/null
trace lavad q epochstorage show-epoch-details >/dev/null
# trace lavad q epochstorage show-fixated-params 0 >/dev/null
# trace lavad q epochstorage show-stake-storage 0 >/dev/null

echo "Testing protocol q commands"
trace lavad q protocol params >/dev/null

echo "Testing conflict q commands"
trace lavad q conflict params >/dev/null
trace lavad q conflict list-conflict-vote >/dev/null
trace lavad q conflict provider-conflicts $(lavad keys show servicer1 -a) >/dev/null
trace lavad q conflict consumer-conflicts $(lavad keys show user1 -a) >/dev/null
# trace lavad q conflict show-conflict-vote stam >/dev/null ## canot test that here

echo "Testing downtime q commands"
trace lavad q downtimequery params >/dev/null
trace lavad q downtimequery downtime 10 >/dev/null

echo "Proposing specs"
(trace lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/ibc.json,./cookbook/specs/cosmoswasm.json,./cookbook/specs/tendermint.json,./cookbook/specs/cosmossdk.json,./cookbook/specs/cosmossdk_45.json,./cookbook/specs/cosmossdk_full.json,./cookbook/specs/ethermint.json,./cookbook/specs/ethereum.json,./cookbook/specs/cosmoshub.json,./cookbook/specs/lava.json,./cookbook/specs/osmosis.json,./cookbook/specs/fantom.json,./cookbook/specs/celo.json,./cookbook/specs/optimism.json,./cookbook/specs/arbitrum.json,./cookbook/specs/starknet.json,./cookbook/specs/aptos.json,./cookbook/specs/juno.json,./cookbook/specs/polygon.json,./cookbook/specs/evmos.json,./cookbook/specs/base.json,./cookbook/specs/canto.json,./cookbook/specs/sui.json,./cookbook/specs/solana.json,./cookbook/specs/bsc.json,./cookbook/specs/axelar.json,./cookbook/specs/avalanche.json,./cookbook/specs/fvm.json,./cookbook/specs/near.json $txoptions) >/dev/null 
wait_count_blocks 2 >/dev/null
(lavad tx gov vote $(latest_vote) yes $txoptions) >/dev/null 
wait_count_blocks 2 >/dev/null 

echo "Testing spec q commands"
trace lavad q spec list-spec >/dev/null
trace lavad q spec show-all-chains >/dev/null
trace lavad q spec show-chain-info ETH1 >/dev/null
trace lavad q spec show-spec ETH1 >/dev/null
 
echo "Proposing plans"
(trace lavad tx gov submit-legacy-proposal plans-add ./cookbook/plans/test_plans/default.json,./cookbook/plans/test_plans/temporary-add.json $txoptions) >/dev/null 
wait_count_blocks 1 >/dev/null
(lavad tx gov vote $(latest_vote) yes $txoptions) >/dev/null 
wait_count_blocks 3 >/dev/null
(trace lavad tx gov submit-legacy-proposal plans-del ./cookbook/plans/test_plans/temporary-del.json $txoptions)>/dev/null 
wait_count_blocks 1 >/dev/null
(lavad tx gov vote $(latest_vote) yes $txoptions) >/dev/null 
wait_count_blocks 2 >/dev/null

echo "Testing plans q commands"
trace lavad q plan params >/dev/null
trace lavad q plan list >/dev/null
trace lavad q plan info DefaultPlan >/dev/null



echo "Testing subscription tx commands"
(trace lavad tx subscription buy DefaultPlan $(lavad keys show alice -a) $txoptions)>/dev/null 
wait_count_blocks 2 >/dev/null
(trace lavad tx subscription add-project myproject --policy-file ./cookbook/projects/example_policy.yml --project-keys-file ./cookbook/projects/example_project_keys.yml --disable $txoptions)>/dev/null 
wait_count_blocks 2 >/dev/null
(trace lavad tx subscription del-project myproject $txoptions)>/dev/null 

echo "Testing subscription q commands"
trace lavad q subscription params >/dev/null
trace lavad q subscription list >/dev/null
trace lavad q subscription current $(lavad keys show alice -a)-admin >/dev/null
trace lavad q subscription list-projects $(lavad keys show alice -a) >/dev/null
trace lavad q subscription next-to-month-expiry >/dev/null

sleep_until_next_epoch >/dev/null

echo "Testing project tx commands"
(lavad tx project set-policy $(lavad keys show alice -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml $txoptions)>/dev/null 
wait_count_blocks 1 >/dev/null
(lavad tx project set-subscription-policy $(lavad keys show alice -a)-admin ./cookbook/projects/policy_all_chains_with_addon.yml $txoptions)>/dev/null 
wait_count_blocks 1 >/dev/null
(trace lavad tx project add-keys $(lavad keys show alice -a)-admin cookbook/projects/example_project_keys.yml $txoptions)>/dev/null 
wait_count_blocks 1 >/dev/null
(lavad tx project del-keys $(lavad keys show alice -a)-admin cookbook/projects/example_project_keys.yml $txoptions)>/dev/null

echo "Testing project q commands"
trace lavad q project params >/dev/null
trace lavad q project info $(lavad keys show alice -a)-admin >/dev/null
trace lavad q project developer $(lavad keys show alice -a) >/dev/null


echo "Testing pairing tx commands"
PROVIDERSTAKE="500000000000ulava"
PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"
wait_count_blocks 1 >/dev/null
(trace lavad tx pairing stake-provider ETH1 $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) --provider-moniker "provider" $txoptions)>/dev/null
wait_count_blocks 1 >/dev/null

CHAINS="SEP1,OSMOSIS,FTM250,CELO,LAV1,OSMOSIST,ALFAJORES,ARB1,ARBN,APT1,STRK,JUN1,COSMOSHUB,POLYGON1,EVMOS,OPTM,BASET,CANTO,SUIT,SOLANA,BSC,AXELAR,AVAX,FVM,NEAR"
(trace lavad tx pairing bulk-stake-provider $CHAINS $PROVIDERSTAKE "$PROVIDER1_LISTENER,1" 1 $(operator_address) --provider-moniker "provider" $txoptions)>/dev/null

sleep_until_next_epoch >/dev/null
(trace lavad tx pairing modify-provider ETH1 --provider-moniker "provider" --delegate-commission 20 --delegate-limit 1000ulava --amount $PROVIDERSTAKE --endpoints "127.0.0.2:2222,1" $txoptions)>/dev/null
wait_count_blocks 1 >/dev/null
(trace lavad tx pairing freeze ETH1,CELO $txoptions)>/dev/null
wait_count_blocks 1 >/dev/null
(trace lavad tx pairing unstake-provider LAV1,OSMOSIST $txoptions)>/dev/null
sleep_until_next_epoch >/dev/null
(trace lavad tx pairing unfreeze ETH1,CELO $txoptions)>/dev/null

echo "Testing pairing q commands"
trace lavad q pairing params >/dev/null
trace lavad q pairing account-info $(lavad keys show alice -a) >/dev/null
trace lavad q pairing effective-policy ETH1 $(lavad keys show alice -a) >/dev/null
trace lavad q pairing get-pairing STRK $(lavad keys show alice -a) >/dev/null
trace lavad q pairing provider-epoch-cu $(lavad keys show servicer1) >/dev/null
trace lavad q pairing providers STRK >/dev/null
trace lavad q pairing provider $(lavad keys show servicer1 -a) STRK >/dev/null
trace lavad q pairing sdk-pairing STRK $(lavad keys show alice -a) >/dev/null
trace lavad q pairing provider-monthly-payout $(lavad keys show servicer1 -a) >/dev/null
trace lavad q pairing subscription-monthly-payout $(lavad keys show user1 -a) >/dev/null
# trace lavad q pairing show-epoch-payments >/dev/null
# trace lavad q pairing show-provider-payment-storage >/dev/null
# trace lavad q pairing show-unique-payment-storage-client-provider >/dev/null
trace lavad q pairing static-providers-list LAV1 >/dev/null
trace lavad q pairing user-entry $(lavad keys show alice -a) ETH1 20 >/dev/null
trace lavad q pairing verify-pairing STRK $(lavad keys show alice -a) $(lavad keys show alice -a) 60 >/dev/null
trace lavad q pairing provider-pairing-chance $(lavad keys show servicer1 -a) STRK 1 "" >/dev/null

echo "Testing dualstaking tx commands"
wait_count_blocks 1 >/dev/null
(trace lavad tx dualstaking delegate $(lavad keys show alice -a) ETH1 $PROVIDERSTAKE $txoptions) >/dev/null
wait_count_blocks 1 >/dev/null
(trace lavad tx dualstaking redelegate $(lavad keys show alice -a) ETH1 $(lavad keys show alice -a) STRK $PROVIDERSTAKE $txoptions) >/dev/null
wait_count_blocks 2 >/dev/null
(trace lavad tx dualstaking claim-rewards $txoptions) >/dev/null
wait_count_blocks 1 >/dev/null
(trace lavad tx dualstaking unbond $(lavad keys show alice -a) STRK $PROVIDERSTAKE $txoptions) >/dev/null

echo "Testing dualstaking q commands"
trace lavad q dualstaking params >/dev/null
trace lavad q dualstaking delegator-providers $(lavad keys show alice -a)>/dev/null
trace lavad q dualstaking delegator-rewards $(lavad keys show alice -a) >/dev/null
trace lavad q dualstaking provider-delegators $(lavad keys show alice -a)>/dev/null

echo "Testing fixationstore q commands"
trace lavad q fixationstore all-indices subscription subs-fs >/dev/null
trace lavad q fixationstore store-keys >/dev/null
trace lavad q fixationstore versions subscription subs-fs $(lavad keys show alice -a) >/dev/null
trace lavad q fixationstore versions entry subs-fs $(lavad keys show alice -a) 100 >/dev/null

echo "Testing rewards q commands"
trace lavad q rewards pools >/dev/null
trace lavad q rewards block-reward >/dev/null
trace lavad q rewards show-iprpc-data > /dev/null
trace lavad q rewards iprpc-provider-reward > /dev/null
trace lavad q rewards iprpc-spec-reward > /dev/null
trace lavad q rewards provider-reward >/dev/null
trace lavad q rewards pending-ibc-iprpc-funds > /dev/null

echo "Testing rewards tx commands"
trace lavad tx rewards fund-iprpc ETH1 4 100000ulava --from alice >/dev/null
wait_count_blocks 1 >/dev/null
trace lavad tx rewards submit-ibc-iprpc-tx ETH1 3 100ulava transfer channel-0 --from bob --home ~/.lava --node tcp://localhost:26657 >/dev/null


echo "Testing events command"
trace lavad test events 30 10 --event lava_relay_payment --from alice --timeout 1s >/dev/null

killall lavad
echo "Testing done :)"