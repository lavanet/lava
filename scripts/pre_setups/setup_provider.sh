#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. "${__dir}"/../vars/variables.sh
LOGS_DIR=${__dir}/../../testutil/debugging/logs
mkdir -p $LOGS_DIR
rm $LOGS_DIR/*.log

echo "---------------Setup Providers------------------"
killall screen
screen -wipe

#ETH providers
screen -d -m -S eth1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ETH1_2221.log" && sleep 0.25

#GTH providers
screen -d -m -S gth_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2121 $GTH_RPC_WS GTH1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/GTH1_2121.log" && sleep 0.25

#FTM providers
screen -d -m -S ftm250_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2251 $FTM_RPC_HTTP FTM250 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/FTM250_2251.log" && sleep 0.25

#Celo providers
screen -d -m -S celo_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 5241 $CELO_HTTP CELO jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/CELO_2221.log" && sleep 0.25

# #Celo alfahores providers
screen -d -m -S alfajores_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 6241 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ALFAJORES_2221.log" && sleep 0.25

#Arbitrum providers
screen -d -m -S arb_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 7241 $ARB1_HTTP ARB1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/ARB1_2221.log" && sleep 0.25

#Aptos providers 
screen -d -m -S apt1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 10031 $APTOS_REST APT1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/APT1_10031.log" && sleep 0.25

#Starknet providers
screen -d -m -S strk_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 8241 $STARKNET_RPC STRK jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/STRK_2221.log"

#Polygon providers
screen -d -m -S polygon_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 4344 $POLYGON_MAINNET_RPC POLYGON1 jsonrpc $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/POLYGON_4344.log"

# All Cosmos-SDK Chains below

# Osmosis providers
screen -d -m -S cos3_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS3_2231.log" && sleep 0.25

# Osmosis testnet providers
screen -d -m -S cos4_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 4231 $OSMO_TEST_REST COS4 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS4_4231.log" && sleep 0.25

# Lava providers
screen -d -m -S lav1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2271 $LAVA_REST LAV1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/LAV1_2271.log" && sleep 0.25

# Cosmoshub providers
screen -d -m -S cos5_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2331 $GAIA_REST COS5 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/COS5_2331.log"

# Juno providers
screen -d -m -S jun1_providers bash -c "source ~/.bashrc; lavad server 127.0.0.1 2371 $JUNO_REST JUN1 rest $EXTRA_PROVIDER_FLAGS --geolocation 1 --log_level debug --from servicer1 2>&1 | tee $LOGS_DIR/JUN1_2371.log"

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
# Cosmos-SDK based chains
screen -S portals -X screen -t win1  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3334 COS3 rest 127.0.0.1:3335 COS3 tendermintrpc 127.0.0.1:3353 COS3 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_COS3_3334.log"
screen -S portals -X screen -t win4  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3337 COS4 rest 127.0.0.1:3338 COS4 tendermintrpc 127.0.0.1:3354 COS4 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_COS4_3337.log"
screen -S portals -X screen -t win7  -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3340 LAV1 rest 127.0.0.1:3341 LAV1 tendermintrpc 127.0.0.1:3352 LAV1 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_LAV1_3340.log"
screen -S portals -X screen -t win10 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3343 COS5 rest 127.0.0.1:3344 COS5 tendermintrpc 127.0.0.1:3356 COS5 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3343.log"
screen -S portals -X screen -t win16 -X bash -c "source ~/.bashrc; lavad rpcconsumer 127.0.0.1:3349 JUN1 rest 127.0.0.1:3350 JUN1 tendermintrpc 127.0.0.1:3355 JUN1 grpc $EXTRA_PORTAL_FLAGS --geolocation 1 --log_level debug --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3349.log"




echo "--- setting up screens done ---"
screen -ls