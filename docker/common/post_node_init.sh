#!/bin/sh
set -e
set -o pipefail

operator_address() {
    lavad q staking validators -o json | jq -r '.validators[0].operator_address'
}

check_votes_status() {
  lavad q gov proposals --output json | jq -r '.proposals[] | select(.status == "PROPOSAL_STATUS_VOTING_PERIOD")' 
}

vote_yes_on_all_pending_proposals() {
  echo "Waiting for at least one proposal to be active"
  while true; do
    sleep 1
    res=$(check_votes_status)
    if [ -n $res ]; then
      echo "No active proposals yet"
    else
      echo "Found active proposal!"
      lavad q gov proposals --output json | jq -r '.proposals[] | select(.status == "PROPOSAL_STATUS_VOTING_PERIOD") | .id' | while read -r proposal_id; do
        echo "$FROM voting yes on proposal id: $proposal_id"
        lavad tx gov vote $proposal_id yes -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE > /dev/null &
      done
      lavad q gov proposals --output json | jq -r '.proposals[] | select(.status == "PROPOSAL_STATUS_VOTING_PERIOD") | .id' | while read -r proposal_id; do
        echo "Waiting for proposal id: $proposal_id to pass..."
        until [ "$(lavad q gov proposal $proposal_id --output json | jq -r .status)" == "PROPOSAL_STATUS_PASSED" ]; do
          echo "Proposal id: $proposal_id didn't pass yet..."
          sleep 1
        done
        echo "Proposal id: $proposal_id finally passed!"
      done
      break
    fi
  done
}

echo "### Starting post node init ###"
wget -O jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64
chmod +x ./jq
mv jq /usr/bin

GASPRICE="${GASPRICE:-0.000000001ulava}"
FROM="${FROM:-servicer1}"
NODE="${NODE:-tcp://lava-node:26657}"

lavad config node $NODE
(
cd /lava/cookbook/specs/
lavad tx gov submit-legacy-proposal spec-add \
  ./ibc.json,./tendermint.json,./cosmoswasm.json,./cosmossdk.json,./cosmossdk_45.json,./cosmossdk_full.json,./ethermint.json,./ethereum.json,./cosmoshub.json,./lava.json \
  --lava-dev-test -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
)
vote_yes_on_all_pending_proposals

echo "Adding plan: DefaultPlan"
lavad tx gov submit-legacy-proposal plans-add /lava/cookbook/plans/test_plans/default.json -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
vote_yes_on_all_pending_proposals

echo "Buying plan: DefaultPlan for $FROM"
lavad tx subscription buy DefaultPlan $(lavad keys show $FROM -a) --enable-auto-renewal -y --from $FROM --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE 2> /dev/null

sleep 1
echo "Staking provider"
PROVIDERSTAKE="500000000000ulava"
PROVIDER_ADDRESS="nginx:80"
lavad tx pairing stake-provider LAV1 $PROVIDERSTAKE "$PROVIDER_ADDRESS,1" 1 $(operator_address) -y --delegate-commission 50  --from servicer1 --provider-moniker "servicer1" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

echo "### Post node init finished successfully ###"