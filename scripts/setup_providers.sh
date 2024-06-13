#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh
LOGS_DIR=${__dir}/../testutil/debugging/logs
# mkdir -p $LOGS_DIR
# rm $LOGS_DIR/*.log

# echo "---------------Setup Providers------------------"
# killall screen
# screen -wipe

EXTRA_PROVIDER_FLAGS="$EXTRA_PROVIDER_FLAGS --chain-id=lava"
EXTRA_PORTAL_FLAGS="$EXTRA_PORTAL_FLAGS --chain-id=lava"
GEOLOCATION=2

PROVIDER1_LISTENER="127.0.0.1:2221"

# echo; echo "#### Starting provider 1 ####"
screen -d -m -S provider1 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$PROVIDER1_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$EXTRA_PROVIDER_FLAGS --metrics-listen-address ":7780" --geolocation "$GEOLOCATION" --log_level trace --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
# $PROVIDER1_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \


echo; echo "#### Starting consumer ####"
# Setup Portal
screen -d -m -S portal1 bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/lava_tendermint_only_consumer_example1.yml\
$EXTRA_PORTAL_FLAGS --geolocation "$GEOLOCATION" --debug-relays --log_level trace --from user1 --chain-id lava --allow-insecure-provider-dialing --strategy distributed 2>&1 | tee $LOGS_DIR/PORTAL1.log" && sleep 0.25

screen -d -m -S portal2 bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/lava_tendermint_only_consumer_example2.yml\
$EXTRA_PORTAL_FLAGS --geolocation "$GEOLOCATION" --debug-relays --log_level trace --from user2 --chain-id lava --allow-insecure-provider-dialing --strategy distributed 2>&1 | tee $LOGS_DIR/PORTAL2.log" && sleep 0.25

# echo "--- setting up screens done ---"
# screen -ls
# echo "ETH1 listening on 127.0.0.1:3333"
# echo "LAV1 listening on 127.0.0.1:3360 - rest 127.0.0.1:3361 - tendermintpc 127.0.0.1:3362 - grpc"