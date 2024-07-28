#!/bin/sh
set -e
set -o pipefail

latest_vote() {
  lavad q gov proposals | grep -o 'id: "[0-9]*"' | cut -d'"' -f2 | tail -n 1
}

operator_address() {
    lavad q staking validators -o json | grep -o 'operator_address: ".*"'
}

wait_next_block() {
  current=$(lavad q block | grep -o '"height":"[0-9]*"' | cut -d':' -f2)
  echo "Waiting for the next block $current"
  while true; do
    new=$(lavad q block | grep -o '"height":"[0-9]*"' | cut -d':' -f2)
    if [ "$current" != "$new" ]; then
      echo "Finished waiting at block $new"
      break
    fi
    sleep 0.5
  done
}

wait_count_blocks() {
  for i in $(seq 1 $1); do
    wait_next_block
  done
}

vote() {
  vote_id="$1"
  echo "Voting yes on vote with id: $vote_id"
  until lavad tx gov vote $vote_id yes -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE; do
    echo "Failed to vote, exit-code: $?. Retrying..."
    sleep 0.1
  done
}

GASPRICE="0.000000001ulava"
FROM="servicer1"
NODE="tcp://lava-node:26657"

lavad config node $NODE
(
cd /lava/cookbook/specs/
lavad tx gov submit-legacy-proposal spec-add \
  ./ibc.json,./tendermint.json,./cosmoswasm.json,./cosmossdk.json,./cosmossdk_45.json,./cosmossdk_full.json,./ethermint.json,./ethereum.json,./cosmoshub.json,./lava.json \
  --lava-dev-test -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
)
specs_vote="$(latest_vote)"

lavad q gov proposal "$specs_vote"

vote "$specs_vote"

# wait_count_blocks 4
# lavad tx gov submit-legacy-proposal plans-add /lava/cookbook/plans/test_plans/default.json -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# wait_count_blocks 2
# plans_vote=$(latest_vote)
# echo "### vote on plans: $plans_vote ###"
# lavad tx gov vote $plans_vote yes -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# echo "#########"
# wait_count_blocks 2
# echo "### plans ###"
# lavad q plan list
# echo "#########"
# # lavad tx subscription buy DefaultPlan $(lavad keys show $FROM -a) --enable-auto-renewal -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
# lavad tx subscription buy DefaultPlan $(lavad keys show user1 -a) --enable-auto-renewal -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# # PROVIDERSTAKE="500000000000ulava"
# # PROVIDER_ADDRESS="provider:2220"
# # lavad tx pairing stake-provider LAV1 $PROVIDERSTAKE "$NGINX_LISTENER,1" 1 $(operator_address) -y --delegate-commission 50 --delegate-limit $PROVIDERSTAKE --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE