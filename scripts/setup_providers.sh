#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh
LOGS_DIR=${__dir}/../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

echo "---------------Setup Providers------------------"
killall screen
screen -wipe

EXTRA_PROVIDER_FLAGS="$EXTRA_PROVIDER_FLAGS --chain-id=lava"
EXTRA_PORTAL_FLAGS="$EXTRA_PORTAL_FLAGS --chain-id=lava"

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"

#ETH providers
screen -d -m -S provider1 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --metrics-listen-address ":7780" --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
# $PROVIDER1_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \
screen -d -m -S provider2 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER2_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER2_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25
# $PROVIDER2_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

screen -d -m -S provider3 bash -c "source ~/.bashrc; lava-protocol rpcprovider \
$PROVIDER3_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER3_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25
# $PROVIDER3_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

# Setup Portal
screen -d -m -S portals bash -c "source ~/.bashrc; lava-protocol rpcconsumer \
127.0.0.1:3360 LAV1 rest 127.0.0.1:3361 LAV1 tendermintrpc 127.0.0.1:3362 LAV1 grpc \
$EXTRA_PORTAL_FLAGS --metrics-listen-address ":7779" --geolocation 1 --log_level debug --from user1 --allow-insecure-provider-dialing 2>&1 | tee $LOGS_DIR/PORTAL.log" && sleep 0.25
# 127.0.0.1:3385 MANTLE jsonrpc \

echo "--- setting up screens done ---"
screen -ls
