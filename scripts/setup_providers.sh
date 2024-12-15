#!/bin/bash
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh
source $__dir/useful_commands.sh
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
$PROVIDER1_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER1_LISTENER SEP1 jsonrpc '$SEP_RPC_WS' \
$PROVIDER1_LISTENER HOL1 jsonrpc '$HOL_RPC_WS' \
$PROVIDER1_LISTENER FTM250 jsonrpc '$FTM_RPC_HTTP' \
$PROVIDER1_LISTENER CELO jsonrpc '$CELO_HTTP' \
$PROVIDER1_LISTENER ALFAJORES jsonrpc '$CELO_ALFAJORES_HTTP' \
$PROVIDER1_LISTENER ARB1 jsonrpc '$ARB1_HTTP' \
$PROVIDER1_LISTENER APT1 rest '$APTOS_REST' \
$PROVIDER1_LISTENER STRK jsonrpc '$STARKNET_RPC' \
$PROVIDER1_LISTENER POLYGON1 jsonrpc '$POLYGON_MAINNET_RPC' \
$PROVIDER1_LISTENER OPTM jsonrpc '$OPTIMISM_RPC' \
$PROVIDER1_LISTENER BASE jsonrpc '$BASE_RPC' \
$PROVIDER1_LISTENER BSC jsonrpc '$BSC_RPC' \
$PROVIDER1_LISTENER SOLANA jsonrpc '$SOLANA_RPC' \
$PROVIDER1_LISTENER SUIT jsonrpc '$SUI_RPC' \
$PROVIDER1_LISTENER OSMOSIS rest '$OSMO_REST' \
$PROVIDER1_LISTENER OSMOSIS tendermintrpc '$OSMO_RPC,$OSMO_RPC' \
$PROVIDER1_LISTENER OSMOSIS grpc '$OSMO_GRPC' \
$PROVIDER1_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER1_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER1_LISTENER LAV1 grpc '$LAVA_GRPC' \
$PROVIDER1_LISTENER COSMOSHUB rest '$GAIA_REST' \
$PROVIDER1_LISTENER COSMOSHUB tendermintrpc '$GAIA_RPC,$GAIA_RPC' \
$PROVIDER1_LISTENER COSMOSHUB grpc '$GAIA_GRPC' \
$PROVIDER1_LISTENER JUN1 rest '$JUNO_REST' \
$PROVIDER1_LISTENER JUN1 tendermintrpc '$JUNO_RPC,$JUNO_RPC' \
$PROVIDER1_LISTENER JUN1 grpc '$JUNO_GRPC' \
$PROVIDER1_LISTENER EVMOS jsonrpc '$EVMOS_RPC' \
$PROVIDER1_LISTENER EVMOS tendermintrpc '$EVMOS_TENDERMINTRPC,$EVMOS_TENDERMINTRPC' \
$PROVIDER1_LISTENER EVMOS rest '$EVMOS_REST' \
$PROVIDER1_LISTENER EVMOS grpc '$EVMOS_GRPC' \
$PROVIDER1_LISTENER CANTO jsonrpc '$CANTO_RPC' \
$PROVIDER1_LISTENER CANTO tendermintrpc '$CANTO_TENDERMINT,$CANTO_TENDERMINT' \
$PROVIDER1_LISTENER CANTO rest '$CANTO_REST' \
$PROVIDER1_LISTENER CANTO grpc '$CANTO_GRPC' \
$PROVIDER1_LISTENER AXELAR tendermintrpc '$AXELAR_RPC_HTTP,$AXELAR_RPC_HTTP' \
$PROVIDER1_LISTENER AXELAR rest '$AXELAR_REST' \
$PROVIDER1_LISTENER AXELAR grpc '$AXELAR_GRPC' \
$PROVIDER1_LISTENER AVAX jsonrpc '$AVALANCH_PJRPC' \
$PROVIDER1_LISTENER AVAXT jsonrpc '$AVALANCHT_PJRPC' \
$PROVIDER1_LISTENER FVM jsonrpc '$FVM_JRPC' \
$PROVIDER1_LISTENER NEAR jsonrpc '$NEAR_JRPC' \
$PROVIDER1_LISTENER AGR rest '$AGORIC_REST' \
$PROVIDER1_LISTENER AGR grpc '$AGORIC_GRPC' \
$PROVIDER1_LISTENER KOIIT jsonrpc '$KOIITRPC' \
$PROVIDER1_LISTENER AGR tendermintrpc '$AGORIC_RPC,$AGORIC_RPC' \
$PROVIDER1_LISTENER AGRT rest '$AGORIC_TEST_REST' \
$PROVIDER1_LISTENER AGRT grpc '$AGORIC_TEST_GRPC' \
$PROVIDER1_LISTENER AGRT tendermintrpc '$AGORIC_TEST_RPC,$AGORIC_TEST_RPC' \
$PROVIDER1_LISTENER STRGZ tendermintrpc '$STARGAZE_RPC_HTTP,$STARGAZE_RPC_HTTP' \
$PROVIDER1_LISTENER STRGZ rest '$STARGAZE_REST' \
$PROVIDER1_LISTENER STRGZ grpc '$STARGAZE_GRPC' \
$EXTRA_PROVIDER_FLAGS --metrics-listen-address ":7780" --geolocation "$GEOLOCATION" --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/PROVIDER1.log" && sleep 0.25
# $PROVIDER1_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

echo; echo "#### Starting provider 2 ####"
screen -d -m -S provider2 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER2_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER2_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER2_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER2_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation "$GEOLOCATION" --log_level debug --from servicer2 --chain-id lava 2>&1 | tee $LOGS_DIR/PROVIDER2.log" && sleep 0.25
# $PROVIDER2_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

echo; echo "#### Starting provider 3 ####"
screen -d -m -S provider3 bash -c "source ~/.bashrc; lavap rpcprovider \
$PROVIDER3_LISTENER ETH1 jsonrpc '$ETH_RPC_WS' \
$PROVIDER3_LISTENER LAV1 rest '$LAVA_REST' \
$PROVIDER3_LISTENER LAV1 tendermintrpc '$LAVA_RPC,$LAVA_RPC_WS' \
$PROVIDER3_LISTENER LAV1 grpc '$LAVA_GRPC' \
$EXTRA_PROVIDER_FLAGS --geolocation "$GEOLOCATION" --log_level debug --from servicer3 --chain-id lava 2>&1 | tee $LOGS_DIR/PROVIDER3.log" && sleep 0.25
# $PROVIDER3_LISTENER MANTLE jsonrpc '$MANTLE_JRPC' \

echo; echo "#### Starting consumer ####"
# Setup Consumer
screen -d -m -S portals bash -c "source ~/.bashrc; lavap rpcconsumer consumer_examples/full_consumer_example.yml\
$EXTRA_PORTAL_FLAGS --cache-be "127.0.0.1:7778" --geolocation "$GEOLOCATION" --debug-relays --log_level debug --from user1 --chain-id lava --allow-insecure-provider-dialing --strategy distributed 2>&1 | tee $LOGS_DIR/CONSUMER.log" && sleep 0.25
# 127.0.0.1:3385 MANTLE jsonrpc \

echo "--- setting up screens done ---"
screen -ls
echo "ETH1 listening on 127.0.0.1:3333"
echo "LAV1 listening on 127.0.0.1:3360 - rest 127.0.0.1:3361 - tendermintpc 127.0.0.1:3362 - grpc"

validate_env