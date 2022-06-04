
echo "---------------Setup Providers------------------"
# killall screen
#Eth providers
echo " ::: STARTING ETH PROVIDERS :::"
go run relayer/cmd/relayer/main.go server 127.0.0.1 2221 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer1 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer2 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2223 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer3 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2224 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer4 &
go run relayer/cmd/relayer/main.go server 127.0.0.1 2225 ws://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/eth/ws/ ETH1 jsonrpc --from servicer5 

# Terra providers 
# screen -S providers -X screen -t win3 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer1"
# screen -S providers -X screen -t win4 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer2"
# screen -S providers -X screen -t win5 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/lcd/ COS1 rest --from servicer3"
# screen -S providers -X screen -t win6 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer1"
# screen -S providers -X screen -t win7 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer2"
# screen -S providers -X screen -t win8 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://kololo8ex9:ifififkwqlspAFJIjfdMCsdmasdgAKoakdFOAKSFOakfaSEFkbntb311esad@168.119.211.250/terra/rpc/ COS1 tendermintrpc --from servicer3"

#osmosis providers
# echo " ::: STARTING OSMOSIS PROVIDERS :::"
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2231 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2232 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2233 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rest/ COS3 rest --from servicer3 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2241 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer1 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2242 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer2 &
# go run relayer/cmd/relayer/main.go server 127.0.0.1 2243 http://DiHj7kLfX6hNhECG:66JiESmAke6RHte3AKqDAyt966nNAcm7yMiYbLet@157.90.213.235/rpc/ COS3 tendermintrpc --from servicer3 
echo " ::: providers done! :::"
# screen -d -m -S portals zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3333 ETH1 jsonrpc --from user1"
# screen -S portals -X screen -t win10 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3334 COS3 rest --from user2"
# screen -S portals -X screen -t win11 -X zsh -c "source ~/.zshrc; cd ~/go/lava; go run relayer/cmd/relayer/main.go portal_server 127.0.0.1 3335 COS3 tendermintrpc --from user2"
# echo "--- setting up screens done ---"
# screen -ls
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