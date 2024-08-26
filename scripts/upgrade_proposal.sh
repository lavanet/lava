#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh

GASPRICE="0.00002ulava"

if [ -z "$1" ]; then
    versionName="vtest"
else
    versionName="$1"
fi

current=$(current_block)
proposal_block=$((current + 20))

echo "proposal_block $proposal_block"

lavad tx gov submit-legacy-proposal software-upgrade $versionName \
--title "lalatitle" \
--upgrade-height $proposal_block \
--no-validate \
--gas "auto" \
--from alice \
--description "lala" \
--keyring-backend "test" \
--gas-prices "0.00002ulava" \
--gas-adjustment "1.5" \
--deposit 10000000ulava \
--yes

wait_count_blocks 2
echo "latest vote2: $(latest_vote)"
lavad tx gov vote $(latest_vote) yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
