#!/bin/bash 
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source ${__dir}/vars/variables.sh

echo "---------------Setup Providers------------------"

# killall screen
#Eth providers
# echo " ::: STARTING ETH PROVIDERS :::"
# lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1 &
# lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2 &
# lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3 &
# lavad server 127.0.0.1 2224 $ETH_RPC_WS ETH1 jsonrpc --from servicer4 &
# lavad server 127.0.0.1 2225 $ETH_RPC_WS ETH1 jsonrpc --from servicer5 

# Terra providers 
# screen -S providers -X screen -t win3 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2231 $TERRA_RPC_LCD COS1 rest --from servicer1"
# screen -S providers -X screen -t win4 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2232 $TERRA_RPC_LCD COS1 rest --from servicer2"
# screen -S providers -X screen -t win5 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2233 $TERRA_RPC_LCD COS1 rest --from servicer3"
# screen -S providers -X screen -t win6 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2241 $TERRA_RPC_TENDERMINT COS1 tendermintrpc --from servicer1"
# screen -S providers -X screen -t win7 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2242 $TERRA_RPC_TENDERMINT COS1 tendermintrpc --from servicer2"
# screen -S providers -X screen -t win8 -X zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2243 $TERRA_RPC_TENDERMINT COS1 tendermintrpc --from servicer3"

#osmosis providers
echo " ::: STARTING OSMOSIS PROVIDERS :::"
lavad server 127.0.0.1 2231 $OSMO_REST COS3 rest --from servicer1 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2232 $OSMO_REST COS3 rest --from servicer2 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2233 $OSMO_REST COS3 rest --from servicer3 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2241 $OSMO_RPC COS3 tendermintrpc --from servicer1 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2242 $OSMO_RPC COS3 tendermintrpc --from servicer2 --geolocation 1 --log_level debug &
lavad server 127.0.0.1 2243 $OSMO_RPC COS3 tendermintrpc --from servicer3 --geolocation 1 --log_level debug 
echo " ::: providers done! :::"
# screen -d -m -S portals zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
# screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3334 COS3 rest --from user2"
# screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2"
# echo "--- setting up screens done ---"
# screen -ls
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad portal_server 127.0.0.1 3335 COS1 tendermintrpc --from user2"

# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2221 $ETH_RPC_WS ETH1 jsonrpc --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2222 $ETH_RPC_WS ETH1 jsonrpc --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2223 $ETH_RPC_WS ETH1 jsonrpc --from servicer3" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2231 $TERRA_RPC_LCD COS1 rest --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2232 $TERRA_RPC_LCD COS1 rest --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2233 $TERRA_RPC_LCD COS1 rest --from servicer3" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2241 $TERRA_RPC_TENDERMINT COS1 jsonrpc --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2242 $TERRA_RPC_TENDERMINT COS1 jsonrpc --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; lavad server 127.0.0.1 2243 $TERRA_RPC_TENDERMINT COS1 jsonrpc --from servicer3" & /