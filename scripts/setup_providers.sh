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
screen -d -m -S eth1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/ETH1_2221.log" && sleep 0.25
screen -S eth1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/ETH1_2222.log"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/ETH1_2223.log"
screen -S eth1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4 2>&1 | tee $LOGS_DIR/ETH1_2224.log"
screen -S eth1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5 2>&1 | tee $LOGS_DIR/ETH1_2225.log"

#GTH providers
screen -d -m -S gth_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2121 $GTH_RPC_WS GTH1 jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/GTH1_2121.log" && sleep 0.25
screen -S gth_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2122 $GTH_RPC_WS GTH1 jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/GTH1_2122.log"
screen -S gth_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2123 $GTH_RPC_WS GTH1 jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/GTH1_2123.log"
screen -S gth_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2124 $GTH_RPC_WS GTH1 jsonrpc --from servicer4 2>&1 | tee $LOGS_DIR/GTH1_2124.log"
screen -S gth_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2125 $GTH_RPC_WS GTH1 jsonrpc --from servicer5 2>&1 | tee $LOGS_DIR/GTH1_2125.log"

#osmosis providers
screen -d -m -S cos3_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest --from servicer1 2>&1 | tee $LOGS_DIR/COS3_2231.log" && sleep 0.25
screen -S cos3_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest --from servicer2 2>&1 | tee $LOGS_DIR/COS3_2232.log"
screen -S cos3_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest --from servicer3 2>&1 | tee $LOGS_DIR/COS3_2233.log"
screen -S cos3_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc --from servicer1 2>&1 | tee $LOGS_DIR/COS3_2241.log"
screen -S cos3_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc --from servicer2 2>&1 | tee $LOGS_DIR/COS3_2242.log"
screen -S cos3_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc --from servicer3 2>&1 | tee $LOGS_DIR/COS3_2243.log"

#FTM providers
screen -d -m -S ftm250_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2251 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/FTM250_2251.log" && sleep 0.25
screen -S ftm250_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2252 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/FTM250_2252.log"
screen -S ftm250_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2253 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/FTM250_2253.log"
screen -S ftm250_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2254 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer4 2>&1 | tee $LOGS_DIR/FTM250_2254.log"
screen -S ftm250_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2255 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer5 2>&1 | tee $LOGS_DIR/FTM250_2255.log"

#osmosis testnet providers
screen -d -m -S cos4_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4231 $OSMO_TEST_REST COS4 rest --from servicer1 2>&1 | tee $LOGS_DIR/COS4_4231.log" && sleep 0.25
screen -S cos4_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4232 $OSMO_TEST_REST COS4 rest --from servicer2 2>&1 | tee $LOGS_DIR/COS4_4232.log"
screen -S cos4_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4233 $OSMO_TEST_REST COS4 rest --from servicer3 2>&1 | tee $LOGS_DIR/COS4_4233.log"
screen -S cos4_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4241 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer1 2>&1 | tee $LOGS_DIR/COS4_4241.log"
screen -S cos4_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4242 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer2 2>&1 | tee $LOGS_DIR/COS4_4242.log"
screen -S cos4_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4243 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer3 2>&1 | tee $LOGS_DIR/COS4_4243.log"

# Lava providers
screen -d -m -S lav1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2271 $LAVA_REST LAV1 rest --from servicer1 2>&1 | tee $LOGS_DIR/LAV1_2271.log" && sleep 0.25
screen -S lav1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2272 $LAVA_REST LAV1 rest --from servicer2 2>&1 | tee $LOGS_DIR/LAV1_2272.log"
screen -S lav1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2273 $LAVA_REST LAV1 rest --from servicer3 2>&1 | tee $LOGS_DIR/LAV1_2273.log"
screen -S lav1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2261 $LAVA_RPC LAV1 tendermintrpc --from servicer1 2>&1 | tee $LOGS_DIR/LAV1_2261.log"
screen -S lav1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2262 $LAVA_RPC LAV1 tendermintrpc --from servicer2 2>&1 | tee $LOGS_DIR/LAV1_2262.log"
screen -S lav1_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2263 $LAVA_RPC LAV1 tendermintrpc --from servicer3 2>&1 | tee $LOGS_DIR/LAV1_2263.log"

#Celo providers
screen -d -m -S celo_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 5241 $CELO_HTTP CELO jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/CELO_2221.log" && sleep 0.25
screen -S celo_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 5242 $CELO_HTTP CELO jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/CELO_2222.log"
screen -S celo_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 5243 $CELO_HTTP CELO jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/CELO_2223.log"

# #Celo alfahores providers
screen -d -m -S alfajores_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 6241 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/ALFAJORES_2221.log" && sleep 0.25
screen -S alfajores_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 6242 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/ALFAJORES_2222.log"
screen -S alfajores_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 6243 $CELO_ALFAJORES_HTTP ALFAJORES jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/ALFAJORES_2223.log"

#Arbitrum providers
screen -d -m -S arb_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 7241 $ARB1_HTTP ARB1 jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/ARB1_2221.log" && sleep 0.25
screen -S arb_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 7242 $ARB1_HTTP ARB1 jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/ARB1_2222.log"
screen -S arb_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 7243 $ARB1_HTTP ARB1 jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/ARB1_2223.log"

#Aptos providers 
screen -d -m -S apt1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 10031 $APTOS_REST APT1 rest --from servicer1 2>&1 | tee $LOGS_DIR/APT1_10031.log" && sleep 0.25
screen -S apt1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 10032 $APTOS_REST APT1 rest --from servicer2 2>&1 | tee $LOGS_DIR/APT1_10032.log"
screen -S apt1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 10033 $APTOS_REST APT1 rest --from servicer3 2>&1 | tee $LOGS_DIR/APT1_10033.log"

#Starknet providers
screen -d -m -S strk_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 8241 $STARKNET_RPC STRK jsonrpc --from servicer1 2>&1 | tee $LOGS_DIR/STRK_2221.log"
screen -S strk_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 8242 $STARKNET_RPC STRK jsonrpc --from servicer2 2>&1 | tee $LOGS_DIR/STRK_2222.log"
screen -S strk_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 8243 $STARKNET_RPC STRK jsonrpc --from servicer3 2>&1 | tee $LOGS_DIR/STRK_2223.log"


# Juno providers
screen -d -m -S jun1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2371 $JUNO_REST JUN1 rest --from servicer1 2>&1 | tee $LOGS_DIR/JUN1_2371.log"
screen -S jun1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2372 $JUNO_REST JUN1 rest --from servicer2 2>&1 | tee $LOGS_DIR/JUN1_2372.log"
screen -S jun1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2373 $JUNO_REST JUN1 rest --from servicer3 2>&1 | tee $LOGS_DIR/JUN1_2373.log"
screen -S jun1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2361 $JUNO_RPC JUN1 tendermintrpc --from servicer1 2>&1 | tee $LOGS_DIR/JUN1_2361.log"
screen -S jun1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2362 $JUNO_RPC JUN1 tendermintrpc --from servicer2 2>&1 | tee $LOGS_DIR/JUN1_2362.log"
screen -S jun1_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2363 $JUNO_RPC JUN1 tendermintrpc --from servicer3 2>&1 | tee $LOGS_DIR/JUN1_2363.log"

# Setup Portals
screen -d -m -S portals zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3333.log" && sleep 0.25
screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3334 COS3 rest --from user2 2>&1 | tee $LOGS_DIR/PORTAL_3334.log"
screen -S portals -X screen -t win12 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2 2>&1 | tee $LOGS_DIR/PORTAL_3335.log"
screen -S portals -X screen -t win13 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3336 FTM250 jsonrpc --from user3 2>&1 | tee $LOGS_DIR/PORTAL_3336.log"
screen -S portals -X screen -t win14 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3337 COS4 rest --from user2 2>&1 | tee $LOGS_DIR/PORTAL_3337.log"
screen -S portals -X screen -t win15 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3338 COS4 tendermintrpc --from user2 2>&1 | tee $LOGS_DIR/PORTAL_3338.log"
screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3339 GTH1 jsonrpc --from user1 2>&1 | tee $LOGS_DIR/PORTAL_3339.log"
screen -S portals -X screen -t win16 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3340 LAV1 rest --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3340.log"
screen -S portals -X screen -t win17 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3341 LAV1 tendermintrpc --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3341.log"
screen -S portals -X screen -t win17 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3342 CELO jsonrpc --from user3 2>&1 | tee $LOGS_DIR/PORTAL_3342.log"
screen -S portals -X screen -t win18 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3342 CELO jsonrpc --from user3 2>&1 | tee $LOGS_DIR/PORTAL_3342.log"
screen -S portals -X screen -t win19 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3343 ALFAJORES jsonrpc --from user3 2>&1 | tee $LOGS_DIR/PORTAL_3343.log"
screen -S portals -X screen -t win20 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3344 ARB1 jsonrpc --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3344.log"
screen -S portals -X screen -t win21 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3345 STRK jsonrpc --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3345.log"
screen -S portals -X screen -t win22 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3346 APT1 rest --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3346.log"
screen -S portals -X screen -t win16 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3343 JUN1 rest --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3343.log"
screen -S portals -X screen -t win17 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3344 JUN1 tendermintrpc --from user4 2>&1 | tee $LOGS_DIR/PORTAL_3344.log"

echo "--- setting up screens done ---"
screen -ls