#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh
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

#GTH providers
screen -d -m -S gth_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2121 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/GTH1_2121.log" && sleep 0.25
screen -S gth_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2122 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/GTH1_2122.log"
screen -S gth_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2123 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/GTH1_2123.log"
screen -S gth_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2124 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer4 2>&1 | tee $LOGS_DIR/GTH1_2124.log"
screen -S gth_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2125 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer5 2>&1 | tee $LOGS_DIR/GTH1_2125.log"


#FTM providers
screen -d -m -S ftm250_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2251 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/FTM250_2251.log" && sleep 0.25
screen -S ftm250_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2252 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/FTM250_2252.log"
screen -S ftm250_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2253 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/FTM250_2253.log"
screen -S ftm250_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2254 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer4 2>&1 | tee $LOGS_DIR/FTM250_2254.log"
screen -S ftm250_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2255 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer5 2>&1 | tee $LOGS_DIR/FTM250_2255.log"

#Celo providers
screen -d -m -S celo_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 5241 $CELO_HTTP CELO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/CELO_2221.log" && sleep 0.25
screen -S celo_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 5242 $CELO_HTTP CELO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/CELO_2222.log"
screen -S celo_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 5243 $CELO_HTTP CELO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/CELO_2223.log"

# #Celo alfahores providers
screen -d -m -S alfajores_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6241 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ALFAJORES_2221.log" && sleep 0.25
screen -S alfajores_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6242 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/ALFAJORES_2222.log"
screen -S alfajores_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6243 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/ALFAJORES_2223.log"

#Arbitrum providers
screen -d -m -S arb_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 7241 $ARB1_HTTP ARB1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ARB1_2221.log" && sleep 0.25
screen -S arb_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 7242 $ARB1_HTTP ARB1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/ARB1_2222.log"
screen -S arb_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 7243 $ARB1_HTTP ARB1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/ARB1_2223.log"

#Aptos providers 
screen -d -m -S apt1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 10031 $APTOS_REST APT1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/APT1_10031.log" && sleep 0.25
screen -S apt1_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 10032 $APTOS_REST APT1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/APT1_10032.log"
screen -S apt1_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 10033 $APTOS_REST APT1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/APT1_10033.log"

#Starknet providers
screen -d -m -S strk_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 8241 $STARKNET_RPC STRK jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/STRK_2221.log"
screen -S strk_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 8242 $STARKNET_RPC STRK jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/STRK_2222.log"
screen -S strk_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 8243 $STARKNET_RPC STRK jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/STRK_2223.log"

#Polygon providers
screen -d -m -S polygon_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 4344 $POLYGON_MAINNET_RPC POLYGON1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/POLYGON_4344.log"
screen -S polygon_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4345 $POLYGON_MAINNET_RPC POLYGON1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/POLYGON_4345.log"
screen -S polygon_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4346 $POLYGON_MAINNET_RPC POLYGON1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/POLYGON_4346.log"

# Optimism providers
screen -d -m -S optimism_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6003 $OPTIMISM_RPC OPTM jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/OPTM_6003.log"
screen -S optimism_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6004 $OPTIMISM_RPC OPTM jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/OPTM_6004.log"
screen -S optimism_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6005 $OPTIMISM_RPC OPTM jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/OPTM_6005.log"

# Base providers
screen -d -m -S base_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6000 $BASE_GOERLI_RPC BASET jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/BASET_6000.log"
screen -S base_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6001 $BASE_GOERLI_RPC BASET jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/BASET_6001.log"
screen -S base_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6002 $BASE_GOERLI_RPC BASET jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/BASET_6002.log"

# SUI providers
screen -d -m -S sui_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6500 $SUI_RPC SUIT jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/SUIT_6500.log"
screen -S sui_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6501 $SUI_RPC SUIT jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/SUIT_6501.log"
screen -S sui_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6502 $SUI_RPC SUIT jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/SUIT_6502.log"

# Cosmos-SDK Chains

# Osmosis providers
screen -d -m -S cos3_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS3_2231.log" && sleep 0.25
screen -S cos3_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS3_2232.log"
screen -S cos3_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS3_2233.log"
screen -S cos3_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $OSMO_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS3_2241.log"
screen -S cos3_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $OSMO_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS3_2242.log"
screen -S cos3_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $OSMO_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS3_2243.log"
screen -S cos3_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2234 $OSMO_GRPC COS3 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS3_2234.log"
screen -S cos3_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2235 $OSMO_GRPC COS3 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS3_2235.log"
screen -S cos3_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2236 $OSMO_GRPC COS3 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS3_2236.log"

# Osmosis testnet providers
screen -d -m -S cos4_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 4231 $OSMO_TEST_REST COS4 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS4_4231.log" && sleep 0.25
screen -S cos4_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4232 $OSMO_TEST_REST COS4 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS4_4232.log"
screen -S cos4_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4233 $OSMO_TEST_REST COS4 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS4_4233.log"
screen -S cos4_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4241 $OSMO_TEST_RPC COS4 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $OSMO_TEST_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS4_4241.log"
screen -S cos4_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4242 $OSMO_TEST_RPC COS4 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $OSMO_TEST_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS4_4242.log"
screen -S cos4_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4243 $OSMO_TEST_RPC COS4 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $OSMO_TEST_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS4_4243.log"
screen -S cos4_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4234 $OSMO_TEST_GRPC COS4 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS4_4234.log"
screen -S cos4_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4235 $OSMO_TEST_GRPC COS4 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS4_4235.log"
screen -S cos4_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4236 $OSMO_TEST_GRPC COS4 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS4_4236.log"

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

# Cosmoshub providers
screen -d -m -S cos5_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2331 $GAIA_REST COS5 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS5_2331.log"
screen -S cos5_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2332 $GAIA_REST COS5 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS5_2332.log"
screen -S cos5_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2333 $GAIA_REST COS5 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS5_2333.log"
screen -S cos5_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2344 $GAIA_RPC COS5 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $GAIA_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS5_2344.log"
screen -S cos5_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2342 $GAIA_RPC COS5 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $GAIA_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS5_2342.log"
screen -S cos5_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2343 $GAIA_RPC COS5 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $GAIA_RPC_HTTP 2>&1 | tee $LOGS_DIR/COS5_2343.log"
screen -S cos5_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2334 $GAIA_GRPC COS5 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS5_2334.log"
screen -S cos5_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2335 $GAIA_GRPC COS5 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/COS5_2335.log"
screen -S cos5_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2336 $GAIA_GRPC COS5 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/COS5_2336.log"

# Juno providers
screen -d -m -S jun1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2371 $JUNO_REST JUN1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/JUN1_2371.log"
screen -S jun1_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2372 $JUNO_REST JUN1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/JUN1_2372.log"
screen -S jun1_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2373 $JUNO_REST JUN1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/JUN1_2373.log"
screen -S jun1_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2361 $JUNO_RPC JUN1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $JUNO_RPC_HTTP 2>&1 | tee $LOGS_DIR/JUN1_2361.log"
screen -S jun1_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2362 $JUNO_RPC JUN1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $JUNO_RPC_HTTP 2>&1 | tee $LOGS_DIR/JUN1_2362.log"
screen -S jun1_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2363 $JUNO_RPC JUN1 tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $JUNO_RPC_HTTP 2>&1 | tee $LOGS_DIR/JUN1_2363.log"
screen -S jun1_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2374 $JUNO_GRPC JUN1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/JUN1_2374.log"
screen -S jun1_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2375 $JUNO_GRPC JUN1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/JUN1_2375.log"
screen -S jun1_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 2376 $JUNO_GRPC JUN1 grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/JUN1_2376.log"

# Evmos providers
screen -d -m -S evmos_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 4347 $EVMOS_RPC EVMOS jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/EVMOS_4347.log"
screen -S evmos_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4351 $EVMOS_RPC EVMOS jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/EVMOS_4351.log"
screen -S evmos_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4355 $EVMOS_RPC EVMOS jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/EVMOS_4355.log"
screen -S evmos_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4348 $EVMOS_TENDERMINTRPC EVMOS tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $EVMOS_TENDERMINTRPC 2>&1 | tee $LOGS_DIR/EVMOS_4348.log"
screen -S evmos_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4352 $EVMOS_TENDERMINTRPC EVMOS tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $EVMOS_TENDERMINTRPC 2>&1 | tee $LOGS_DIR/EVMOS_4352.log"
screen -S evmos_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4356 $EVMOS_TENDERMINTRPC EVMOS tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $EVMOS_TENDERMINTRPC 2>&1 | tee $LOGS_DIR/EVMOS_4356.log"
screen -S evmos_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4349 $EVMOS_REST EVMOS rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/EVMOS_4349.log"
screen -S evmos_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4353 $EVMOS_REST EVMOS rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/EVMOS_4353.log"
screen -S evmos_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4357 $EVMOS_REST EVMOS rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/EVMOS_4357.log"
screen -S evmos_providers -X screen -t win09 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4350 $EVMOS_GRPC EVMOS grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/EVMOS_grpc0.log"
screen -S evmos_providers -X screen -t win10 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4354 $EVMOS_GRPC EVMOS grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/EVMOS_grpc1.log"
screen -S evmos_providers -X screen -t win11 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 4358 $EVMOS_GRPC EVMOS grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/EVMOS_grpc2.log"

# Canto Providers
screen -d -m -S canto_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6006 $CANTO_RPC CANTO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/CANTO_jsonrpc1.log"
screen -S canto_providers -X screen -t win1 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6007 $CANTO_RPC CANTO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/CANTO_jsonrpc2.log"
screen -S canto_providers -X screen -t win2 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6008 $CANTO_RPC CANTO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/CANTO_jsonrpc3.log"
screen -S canto_providers -X screen -t win3 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6009 $CANTO_TENDERMINT CANTO tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 --tendermint-http-endpoint $CANTO_TENDERMINT 2>&1 | tee $LOGS_DIR/CANTO_tender1.log"
screen -S canto_providers -X screen -t win4 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6010 $CANTO_TENDERMINT CANTO tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 --tendermint-http-endpoint $CANTO_TENDERMINT 2>&1 | tee $LOGS_DIR/CANTO_tender2.log"
screen -S canto_providers -X screen -t win5 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6011 $CANTO_TENDERMINT CANTO tendermintrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 --tendermint-http-endpoint $CANTO_TENDERMINT 2>&1 | tee $LOGS_DIR/CANTO_tender3.log"
screen -S canto_providers -X screen -t win6 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6012 $CANTO_REST CANTO rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/CANTO_rest1.log"
screen -S canto_providers -X screen -t win7 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6013 $CANTO_REST CANTO rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/CANTO_rest2.log"
screen -S canto_providers -X screen -t win8 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6014 $CANTO_REST CANTO rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/CANTO_rest3.log"
screen -S canto_providers -X screen -t win09 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6015 $CANTO_GRPC CANTO grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/CANTO_grpc1.log"
screen -S canto_providers -X screen -t win10 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6016 $CANTO_GRPC CANTO grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer2 2>&1 | tee $LOGS_DIR/CANTO_grpc2.log"
screen -S canto_providers -X screen -t win11 -X bash -c "source ~/.bashrc; lavad server 127.0.0.1 6017 $CANTO_GRPC CANTO grpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer3 2>&1 | tee $LOGS_DIR/CANTO_grpc3.log"

# Setup Portals
screen -d -m -S portals bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3333 ETH1 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_ETH_3333.log" && sleep 0.25
screen -S portals -X screen -t win3  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3336 FTM250 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_FTM250_3336.log"
screen -S portals -X screen -t win6  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3339 GTH1 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"
screen -S portals -X screen -t win9  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3342 CELO jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3342.log"
screen -S portals -X screen -t win12 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3345 ALFAJORES jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3345.log"
screen -S portals -X screen -t win13 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3346 ARB1 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3346.log"
screen -S portals -X screen -t win14 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3347 STRK jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3347.log"
screen -S portals -X screen -t win15 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3348 APT1 rest $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3348.log"
screen -S portals -X screen -t win18 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3351 POLYGON1 jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3351.log"
screen -S portals -X screen -t win21 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3362 OPTM jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_OPTM.log"
screen -S portals -X screen -t win20 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3361 BASET jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_BASET.log"
screen -S portals -X screen -t win20 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3364 SUIT jsonrpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_SUIT.log"
# Cosmos-SDK based chains
screen -S portals -X screen -t win1  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3334 COS3 rest 127.0.0.1:3335 COS3 tendermintrpc 127.0.0.1:3353 COS3 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_COS3_3334.log"
screen -S portals -X screen -t win4  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3337 COS4 rest 127.0.0.1:3338 COS4 tendermintrpc 127.0.0.1:3354 COS4 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_COS4_3337.log"
screen -S portals -X screen -t win7  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3340 LAV1 rest 127.0.0.1:3341 LAV1 tendermintrpc 127.0.0.1:3352 LAV1 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_LAV1_3340.log"
screen -S portals -X screen -t win10 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3343 COS5 rest 127.0.0.1:3344 COS5 tendermintrpc 127.0.0.1:3356 COS5 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3343.log"
screen -S portals -X screen -t win16 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3349 JUN1 rest 127.0.0.1:3350 JUN1 tendermintrpc 127.0.0.1:3355 JUN1 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3349.log"
screen -S portals -X screen -t win19 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3360 EVMOS jsonrpc 127.0.0.1:3357 EVMOS rest 127.0.0.1:3358 EVMOS tendermintrpc 127.0.0.1:3359 EVMOS grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_EVMOS.log"
screen -S portals -X screen -t win22 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3363 CANTO jsonrpc 127.0.0.1:3364 CANTO rest 127.0.0.1:3365 CANTO tendermintrpc 127.0.0.1:3366 CANTO grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_CANTO.log"


echo "--- setting up screens done ---"
screen -ls