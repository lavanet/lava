lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice --gas-adjustment "1.5" --gas "auto" -y
wait
lavad tx gov vote 1 yes -y --from alice
sleep 20
lavad tx servicer stake-servicer "Ethereum Mainnet" 2050stake 0 "127.0.0.1:2221" -y --from servicer1
lavad tx servicer stake-servicer "Ethereum Mainnet" 2100stake 0 "127.0.0.1:2222" -y --from servicer2
lavad tx servicer stake-servicer "Ethereum Mainnet" 2200stake 0 "127.0.0.1:2223" -y --from servicer3
lavad tx servicer stake-servicer "Ethereum Mainnet" 2000stake 0 "127.0.0.1:2224" -y --from servicer4
lavad tx servicer stake-servicer "Ethereum Mainnet" 2150stake 0 "127.0.0.1:2225" -y --from servicer5
lavad tx user stake-user "Ethereum Mainnet" 2000stake 0 -y --from user1
echo "---------------Queries------------------"
lavad query servicer staked-servicers "Ethereum Mainnet"
lavad query user staked-users "Ethereum Mainnet"
