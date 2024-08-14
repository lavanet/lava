#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh
. ${__dir}/vars/variables.sh
# Making sure old screens are not running
echo "current vote number $(latest_vote)"
killall screen
screen -wipe
GASPRICE="0.000000001ulava"

for i in $(seq 1 100); do
  users+=("useroren$i")
done

for user in "${users[@]}"; do
  lavad tx pairing unstake-provider ETH1 "$(operator_address)" --from $user -y --gas-adjustment "1.5" --gas "auto" --gas-prices "$GASPRICE"
  echo "Unstake failed for user: $user" >&2
done