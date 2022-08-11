#!/bin/bash
MYDIR="$(dirname "$(realpath "$0")")"
echo "$MYDIR"
source $MYDIR/env_vars_for_upgrade.sh

# maybe we need to
# rm -rf $DAEMON_HOME
ignite chain build
ignite chain init

echo "binary name: $DAEMON_NAME"
echo "cosmovisor path: $DAEMON_NAME"
echo "removing old cosmovisor dir $DAEMON_HOME/cosmovisor"
rm -rf $DAEMON_HOME/cosmovisor
echo "creating $DAEMON_HOME/cosmovisor/genesis/bin"
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
echo "copying /home/user/go/bin/lavad to $DAEMON_HOME/cosmovisor/genesis/bin"
cp $HOME/go/bin/lavad $DAEMON_HOME/cosmovisor/genesis/bin

cosmovisor start start

# option 1. rebuild, put new binary into the cosmovisor directory of upgrades
# option 2. fetch the binary from a server.

# next step:
# run upgrade a running module from a different terminal.



















# an example for a proposal with upgrade binaries with remote path to binary file. (this requires that the wont be a binary file in
# the cosmovisor upgrade path)

#  tx gov submit-proposal software-upgrade Vega \
# --title Vega \
# --deposit 100uatom \
# --upgrade-height 7368420 \
# --upgrade-info '{"binaries":{"linux/amd64":"https://github.com/cosmos/gaia/releases/download/v6.0.0-rc1/gaiad-v6.0.0-rc1-linux-amd64","linux/arm64":"https://github.com/cosmos/gaia/releases/download/v6.0.0-rc1/gaiad-v6.0.0-rc1-linux-arm64","darwin/amd64":"https://github.com/cosmos/gaia/releases/download/v6.0.0-rc1/gaiad-v6.0.0-rc1-darwin-amd64"}}' \
# --description "upgrade to Vega" \
# --gas 400000 \
# --from user \
# --chain-id test \
# --home test/val2 \
# --node tcp://localhost:36657 \
# --yes