#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "$__dir"/../useful_commands.sh
. "${__dir}"/../vars/variables.sh
# Making sure old screens are not running
killall screen
screen -wipe
LOGS_DIR=${__dir}/../../testutil/debugging/logs
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_lava.json,./cookbook/spec_add_ethereum.json,./cookbook/spec_add_osmosis.json,./cookbook/spec_add_fantom.json,./cookbook/spec_add_goerli.json  -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4 
lavad tx pairing stake-client "LAV1" 200000ulava 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "LAV1" 2010ulava "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1 127.0.0.1:2281,grpc,1" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2000ulava "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1 127.0.0.1:2282,grpc,1" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" 2050ulava "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1 127.0.0.1:2283,grpc,1" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

# Lava providers
screen -d -m -S lav1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2271 $LAVA_REST LAV1 rest --from servicer1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2271.log"; sleep 0.3
screen -S lav1_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2272 $LAVA_REST LAV1 rest --from servicer2 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2272.log"
screen -S lav1_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2273 $LAVA_REST LAV1 rest --from servicer3 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2273.log"
screen -S lav1_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2261 $LAVA_RPC LAV1 tendermintrpc --from servicer1$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2261.log"
screen -S lav1_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2262 $LAVA_RPC LAV1 tendermintrpc --from servicer2$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2262.log"
screen -S lav1_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2263 $LAVA_RPC LAV1 tendermintrpc --from servicer3$EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2263.log"
screen -S lav1_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2281 $LAVA_GRPC LAV1 grpc --from servicer1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2281.log"
screen -S lav1_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2282 $LAVA_GRPC LAV1 grpc --from servicer2 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2282.log"
screen -S lav1_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2283 $LAVA_GRPC LAV1 grpc --from servicer3 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug 2>&1 | tee $LOGS_DIR/LAV1_2283.log"

screen -d -m -S portals bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3340 LAV1 rest 127.0.0.1:3341 LAV1 tendermintrpc 127.0.0.1:3342 LAV1 grpc --from user1 $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug | tee $LOGS_DIR/LAV1_tendermint_portal.log"; sleep 0.3
screen -r portals
# Lava Over Lava ETH

sleep 3 # wait for the portal to start.

# lavad tx pairing stake-client "ETH1" 200000ulava 1 -y --from user1 --node "http://127.0.0.1:3341/1"

# lavad tx pairing stake-provider "ETH1" 2010ulava "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --node "http://127.0.0.1:3341/1"

# sleep_until_next_epoch

# screen -d -m -S eth1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 --node \"http://127.0.0.1:3341/1\" 2>&1 | tee $LOGS_DIR/ETH1_2221.log"

# screen -d -m -S eth1_portals bash -c "source ~/.bashrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1 --node \"http://127.0.0.1:3341/1\" 2>&1 | tee $LOGS_DIR/PORTAL_3333.log"

screen -ls