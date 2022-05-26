lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice --gas-adjustment "1.5" --gas "auto" -y
# lavad tx gov submit-proposal spec-add ./cookbook/spec_add_terra.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_osmosis.json --from alice --gas-adjustment "1.5" --gas "auto" -y
wait
lavad tx gov vote 1 yes -y --from alice
lavad tx gov vote 2 yes -y --from alice
# lavad tx gov vote 3 yes -y --from alice
sleep 4

#Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010stake "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1
lavad tx pairing stake-provider "ETH1" 2000stake "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2
lavad tx pairing stake-provider "ETH1" 2050stake "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3
lavad tx pairing stake-provider "ETH1" 2020stake "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4
lavad tx pairing stake-provider "ETH1" 2030stake "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5

#Terra providers
lavad tx pairing stake-provider "COS3" 2010stake "127.0.0.1:2241,tendermintrpc,1 127.0.0.1:2231,rest,1" 1 -y --from servicer1
lavad tx pairing stake-provider "COS3" 2000stake "127.0.0.1:2242,tendermintrpc,1 127.0.0.1:2232,rest,1" 1 -y --from servicer2
lavad tx pairing stake-provider "COS3" 2050stake "127.0.0.1:2243,tendermintrpc,1 127.0.0.1:2233,rest,1" 1 -y --from servicer3
# lavad tx pairing stake-provider "COS1" 2020stake "127.0.0.1:2244,tendermintrpc,1 127.0.0.1:2234,rest,1" 1 -y --from servicer4
# lavad tx pairing stake-provider "COS1" 2030stake "127.0.0.1:2245,tendermintrpc,1 127.0.0.1:2235,rest,1" 1 -y --from servicer5

lavad tx pairing stake-client "ETH1" 200000stake 1 -y --from user1
lavad tx pairing stake-client "COS3" 200000stake 1 -y --from user2

echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"
echo "---------------Setup Providers------------------"
killall screen
#Eth providers
screen -d -m -S eth1_providers zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer1"
screen -S eth1_providers -X screen -t win1 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer2"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer3"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer4"
screen -S eth1_providers -X screen -t win2 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer5"

# Terra providers 
# screen -S providers -X screen -t win3 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer1"
# screen -S providers -X screen -t win4 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer2"
# screen -S providers -X screen -t win5 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer3"
# screen -S providers -X screen -t win6 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer1"
# screen -S providers -X screen -t win7 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer2"
# screen -S providers -X screen -t win8 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer3"

#osmosis providers
screen -d -m -S cos3_providers zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://157.90.213.235:1317/ COS3 rest --from servicer1"
screen -S cos3_providers -X screen -t win4 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://157.90.213.235:1317/ COS3 rest --from servicer2"
screen -S cos3_providers -X screen -t win5 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://157.90.213.235:1317/ COS3 rest --from servicer3"
screen -S cos3_providers -X screen -t win6 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://157.90.213.235:26657/ COS3 tendermintrpc --from servicer1"
screen -S cos3_providers -X screen -t win7 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://157.90.213.235:26657/ COS3 tendermintrpc --from servicer2"
screen -S cos3_providers -X screen -t win8 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://157.90.213.235:26657/ COS3 tendermintrpc --from servicer3"

screen -d -m -S portals zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3334 COS3 rest --from user2"
screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2"
echo "--- setting up screens done ---"
screen -ls
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3335 COS1 tendermintrpc --from user2"

# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer3" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer3" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 jsonrpc --from servicer1" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 jsonrpc --from servicer2" & /
# cmd.exe /c start wsl.exe zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 jsonrpc --from servicer3" & /