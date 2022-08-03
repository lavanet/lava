#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
. ${__dir}/vars/variables.sh

echo "---------------Setup Providers------------------"
killall screen

#ETH providers
screen -d -m -S eth1_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1"
screen -S eth1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3"
screen -S eth1_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4"
screen -S eth1_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5"

#osmosis providers
screen -d -m -S cos3_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest --from servicer1"
screen -S cos3_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest --from servicer2"
screen -S cos3_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest --from servicer3"
screen -S cos3_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc --from servicer1"
screen -S cos3_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc --from servicer2"
screen -S cos3_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc --from servicer3"

#FTM providers
screen -d -m -S ftm250_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2251 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer1"
screen -S ftm250_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2252 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer2"
screen -S ftm250_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2253 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer3"
screen -S ftm250_providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2254 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer4"
screen -S ftm250_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2255 $FTM_RPC_HTTP FTM250 jsonrpc --from servicer5"

#osmosis testnet providers
screen -d -m -S cos4_providers zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4231 $OSMO_TEST_REST COS4 rest --from servicer1"
screen -S cos4_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4232 $OSMO_TEST_REST COS4 rest --from servicer2"
screen -S cos4_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4233 $OSMO_TEST_REST COS4 rest --from servicer3"
screen -S cos4_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4241 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer1"
screen -S cos4_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4242 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer2"
screen -S cos4_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 4243 $OSMO_TEST_RPC COS4 tendermintrpc --from servicer3"

screen -d -m -S portals zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3334 COS3 rest --from user2"
screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2"
screen -S portals -X screen -t win12 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3336 FTM250 jsonrpc --from user3"
screen -S portals -X screen -t win13 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3337 COS4 rest --from user2"
screen -S portals -X screen -t win14 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3338 COS4 tendermintrpc --from user2"
echo "--- setting up screens done ---"
screen -ls