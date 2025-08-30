#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
killall screen
rm -rf ~/.lavavisor
rm -rf ~/.lava

LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

echo "[Lavavisor Setup] installing all binaries"
make install-all 

echo "[Lavavisor Setup] setting up a new lava node"
screen -d -m -S node bash -c "./scripts/start_env_dev.sh"
screen -ls
echo "[Lavavisor Setup] sleeping 20 seconds for node to finish setup (if its not enough increase timeout)"
sleep 20

echo "[Lavavisor Setup] checking node is up"
lavad status

echo "[Lavavisor Setup] initializing lavavisor"
lavavisor init --auto-download; 
sleep 0.5
echo "[Lavavisor Setup] finished initialization"

echo "[Lavavisor Setup] creating service files for lavavisor"
lavavisor create-service provider ./config/provider_examples/lava_example.yml --from servicer1 --geolocation 1 --chain-id lava --keyring-backend test --create-link; sleep 0.5
echo "[Lavavisor Setup] finished creating service files for lavavisor"

echo "[Lavavisor Setup] submitting spec proposal"
specs=$(get_all_specs)
lavad tx gov submit-legacy-proposal spec-add $specs -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava &
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava;

echo "[Lavavisor Setup] adding lavavisor screen"
screen -d -m -S lavavisor bash -c "lavavisor start --auto-download 2>&1 | tee $LOGS_DIR/LAVAVISOR.log";
screen -ls
echo "[Lavavisor Setup] sleeping 10 seconds for lavavisor to finish setup (if its not enough increase timeout)"
sleep 10

echo "[Lavavisor Setup] submitting param change vote"
lavad tx gov submit-legacy-proposal param-change ./cookbook/param_changes/param_change_version_upgrade.json --from alice -y --gas-adjustment 1.5 --gas auto --gas-prices 0.00002ulava; 
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.00002ulava;
