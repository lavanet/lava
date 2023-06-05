#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh
. ${__dir}/vars/variables.sh

# Making sure old screens are not running
killall screen
screen -wipe
GASPRICE="0.000000001ulava"
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmoswasm.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_cosmossdk_full.json,./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_cosmoshub.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_lava.json,./cookbook/specs/spec_add_osmosis.json,./cookbook/specs/spec_add_fantom.json,./cookbook/specs/spec_add_celo.json,./cookbook/specs/spec_add_optimism.json,./cookbook/specs/spec_add_arbitrum.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep 4
lavad tx gov submit-proposal plans-add ./cookbook/specs/plans/default.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

CLIENTSTAKE="500000000000ulava"
PROVIDERSTAKE="500000000000ulava"

sleep 4

lavad tx subscription buy DefaultPlan -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2221,jsonrpc,1" --provider-moniker "dummyMoniker" 1 -y --from servicer1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2222,jsonrpc,1" --provider-moniker "dummyMoniker" 1 -y --from servicer2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2223,jsonrpc,1" --provider-moniker "dummyMoniker" 1 -y --from servicer3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2224,jsonrpc,1" --provider-moniker "dummyMoniker" 1 -y --from servicer4 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $PROVIDERSTAKE "127.0.0.1:2225,jsonrpc,1" --provider-moniker "dummyMoniker" 1 -y --from servicer5 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava Providers
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1 127.0.0.1:2274,grpc,1" 1 -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1 127.0.0.1:2275,grpc,1" 1 -y --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $PROVIDERSTAKE "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1 127.0.0.1:2276,grpc,1" 1 -y --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE


# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch
echo "--done--"

LOGS_DIR=${__dir}/../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

echo "---------------Setup Providers------------------"
killall screen
screen -wipe
#ETH providers
screen -d -m -S eth1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ETH1_2221.log" && sleep 0.25
screen -S eth1_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/ETH1_2222.log"
screen -S eth1_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/ETH1_2223.log"
screen -S eth1_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer4 2>&1 | tee $LOGS_DIR/ETH1_2224.log"
screen -S eth1_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer5 2>&1 | tee $LOGS_DIR/ETH1_2225.log"

# Lava providers
screen -d -m -S lav1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2271 $LAVA_REST LAV1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/LAV1_2271.log" && sleep 0.25
screen -S lav1_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2272 $LAVA_REST LAV1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/LAV1_2272.log"
screen -S lav1_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2273 $LAVA_REST LAV1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/LAV1_2273.log"
screen -S lav1_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2261 $LAVA_RPC LAV1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $LAVA_RPC_HTTP 2>&1 | tee $LOGS_DIR/LAV1_2261.log"
screen -S lav1_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2262 $LAVA_RPC LAV1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $LAVA_RPC_HTTP 2>&1 | tee $LOGS_DIR/LAV1_2262.log"
screen -S lav1_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2263 $LAVA_RPC LAV1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $LAVA_RPC_HTTP 2>&1 | tee $LOGS_DIR/LAV1_2263.log"
screen -S lav1_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2274 $LAVA_GRPC LAV1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/LAV1_2274.log"
screen -S lav1_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2275 $LAVA_GRPC LAV1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/LAV1_2275.log"
screen -S lav1_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2276 $LAVA_GRPC LAV1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/LAV1_2276.log"

# portal
screen -d -m -S portals bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3333 ETH1 jsonrpc 127.0.0.1:3340 LAV1 rest 127.0.0.1:3341 LAV1 tendermintrpc 127.0.0.1:3352 LAV1 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_ALL_SUB.log"

echo "--done setting screens--"
screen -ls