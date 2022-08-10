#!/bin/bash
MYDIR="$(dirname "$(realpath "$0")")"
echo "$MYDIR"
source $MYDIR/env_vars_for_upgrade.sh

git checkout upgrade_module_to

ignite chain build

echo "mkdir -p $DAEMON_HOME/cosmovisor/upgrades/$UPRADE_NAME/bin"
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/$UPRADE_NAME/bin
cp $HOME/go/bin/lavad $DAEMON_HOME/cosmovisor/upgrades/$UPRADE_NAME/bin
echo "cp $HOME/go/bin/lavad $DAEMON_HOME/cosmovisor/upgrades/$UPRADE_NAME/bin"

for i in $(lavad q block | tr "," "\n");
do
    if [[ "$i" == *"height"* ]]; then
        BLOCK_HEIGHT=$(echo $i | sed 's/"//g' | sed 's/height://')
        break
    fi 
done

BLOCK_HEIGHT_CHOSEN=$(echo "$((BLOCK_HEIGHT + 20))")

lavad tx gov submit-proposal software-upgrade $UPRADE_NAME --title upgrade --description upgrade --upgrade-height $BLOCK_HEIGHT_CHOSEN --from alice --yes --gas "auto"
lavad tx gov deposit 1 10000000ulava --from alice --yes --gas "auto"
lavad tx gov vote 1 yes --from alice --yes --gas "auto"
echo "chosen block for upgrade: $BLOCK_HEIGHT_CHOSEN"
lavad q upgrade plan
# wait for upgrade.