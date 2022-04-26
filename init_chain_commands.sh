lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_terra.json --from alice --gas-adjustment "1.5" --gas "auto" -y
wait
lavad tx gov vote 1 yes -y --from alice
lavad tx gov vote 2 yes -y --from alice
sleep 20

#Ethereum providers
lavad tx pairing stake-provider "ETH1" 2010stake "127.0.0.1:2221,json-rpc,1" 1 -y --from servicer1
lavad tx pairing stake-provider "ETH1" 2000stake "127.0.0.1:2222,json-rpc,1" 1 -y --from servicer2
lavad tx pairing stake-provider "ETH1" 2050stake "127.0.0.1:2223,json-rpc,1" 1 -y --from servicer3
lavad tx pairing stake-provider "ETH1" 2020stake "127.0.0.1:2224,json-rpc,1" 1 -y --from servicer4
lavad tx pairing stake-provider "ETH1" 2030stake "127.0.0.1:2225,json-rpc,1" 1 -y --from servicer5

#Terra providers
lavad tx pairing stake-provider "COS1" 2010stake "127.0.0.1:2241,json-rpc,1 127.0.0.1:2231,rest,1" 1 -y --from servicer1
lavad tx pairing stake-provider "COS1" 2000stake "127.0.0.1:2242,json-rpc,1 127.0.0.1:2232,rest,1" 1 -y --from servicer2
lavad tx pairing stake-provider "COS1" 2050stake "127.0.0.1:2243,json-rpc,1 127.0.0.1:2233,rest,1" 1 -y --from servicer3
lavad tx pairing stake-provider "COS1" 2020stake "127.0.0.1:2244,json-rpc,1 127.0.0.1:2234,rest,1" 1 -y --from servicer4
lavad tx pairing stake-provider "COS1" 2030stake "127.0.0.1:2245,json-rpc,1 127.0.0.1:2235,rest,1" 1 -y --from servicer5

lavad tx pairing stake-client "ETH1" 200000stake 1 -y --from user1
lavad tx pairing stake-client "COS1" 200000stake 1 -y --from user2

echo "---------------Queries------------------"
lavad query pairing providers "ETH1"
lavad query pairing clients "ETH1"
