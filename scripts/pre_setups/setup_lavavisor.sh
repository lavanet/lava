#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh

rm -rf ~/.lavavisor
rm -rf ~/.lava

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
lavad tx gov submit-legacy-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_45.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json,./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json,./cookbook/specs/spec_add_starknet.json,./cookbook/specs/spec_add_aptos.json,./cookbook/specs/spec_add_juno.json,./cookbook/specs/spec_add_polygon.json,./cookbook/specs/spec_add_evmos.json,./cookbook/specs/spec_add_base.json,./cookbook/specs/spec_add_canto.json,./cookbook/specs/spec_add_sui.json,./cookbook/specs/spec_add_solana.json,./cookbook/specs/spec_add_bsc.json,./cookbook/specs/spec_add_axelar.json,./cookbook/specs/spec_add_avalanche.json,./cookbook/specs/spec_add_fvm.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE &
wait_next_block
wait_next_block
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava;

echo "[Lavavisor Setup] adding lavavisor screen"
screen -d -m -S lavavisor bash -c "lavavisor start --auto-download";
screen -ls
echo "[Lavavisor Setup] sleeping 60 seconds for lavavisor to finish setup (if its not enough increase timeout)"
sleep 60

echo "[Lavavisor Setup] submitting param change vote"
lavad tx gov submit-legacy-proposal param-change ./param_change_version_upgrade.json --from alice -y --gas-adjustment 1.5 --gas auto --gas-prices 0.000000001ulava; 
wait_next_block
wait_next_block
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices 0.000000001ulava;
