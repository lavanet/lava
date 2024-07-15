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
GEOLOCATION=2

PROVIDER1_LISTENER="127.0.0.1:2221"
PROVIDER2_LISTENER="127.0.0.1:2222"
PROVIDER3_LISTENER="127.0.0.1:2223"

echo; echo "#### Starting cache server for provider ####"
screen -d -m -S cache-provider bash -c "source ~/.bashrc; lavap cache 127.0.0.1:7777 --metrics_address 127.0.0.1:5747 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE_PROVIDER.log"

echo; echo "#### Starting cache server for consumer ####"
screen -d -m -S cache-consumer bash -c "source ~/.bashrc; lavap cache 127.0.0.1:7778 --metrics_address 127.0.0.1:5748 --log_level debug 2>&1 | tee $LOGS_DIR/CACHE_CONSUMER.log"

echo; echo "#### Starting provider 1 ####"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --metrics-listen-address ":7780" --geolocation "$GEOLOCATION" --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
# $PROVIDER1_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

echo; echo "#### Starting consumer ####"
# Setup Portal
screen -d -m -S portals bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/lava_example.yml\
$EXTRA_PORTAL_FLAGS --geolocation "$GEOLOCATION" --debug-relays --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --strategy distributed 2>&1 | tee $LOGS_DIR/PORTAL.log" && sleep 0.25
# 127.0.0.1:3385 MANTLE jsonrpc \

echo "--- setting up screens done ---"
screen -ls
echo "ETH1 listening on 127.0.0.1:3333"
echo "LAV1 listening on 127.0.0.1:3360 - rest 127.0.0.1:3361 - tendermintpc 127.0.0.1:3362 - grpc"