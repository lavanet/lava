#!/bin/sh
set -e
set -o pipefail

latest_vote() {
  lavad q gov proposals | grep -o 'id: "[0-9]*"' | cut -d'"' -f2 | tail -n 1
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
    sleep 1
  done
}

wait_count_blocks() {
  for i in $(seq 1 $1); do
    wait_next_block
  done
}

GASPRICE="0.000000001ulava"
FROM="user1"
NODE="tcp://lava-node:26657"

lavad config node $NODE
(
cd /lava/cookbook/specs/
lavad tx gov submit-legacy-proposal spec-add \
  ./ibc.json,./tendermint.json,./cosmoswasm.json,./cosmossdk.json,./cosmossdk_45.json,./cosmossdk_full.json,./ethermint.json,./ethereum.json,./cosmoshub.json,./lava.json \
  --lava-dev-test -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
)
wait_count_blocks 2
lavad tx gov vote $(latest_vote) yes -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 4
sleep 4
lavad tx gov submit-legacy-proposal plans-add /lava/cookbook/plans/test_plans/default.json -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 2
sleep 4
lavad tx gov vote $(latest_vote) yes -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
wait_count_blocks 4
sleep 4
lavad tx subscription buy DefaultPlan $(lavad keys show $FROM -a) --enable-auto-renewal -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE