lavad tx gov submit-proposal spec-add ./cookbook/spec_add_ethereum.json --from alice --gas-adjustment "1.5" --gas "auto" -y
lavad tx gov submit-proposal spec-add ./cookbook/spec_add_terra.json --from alice --gas-adjustment "1.5" --gas "auto" -y
wait
lavad tx gov vote 1 yes -y --from alice
lavad tx gov vote 2 yes -y --from alice
sleep 20

#Ethereum providers
lavad tx pairing stake-provider "Ethereum Mainnet" 2010stake "127.0.0.1:2221,grpc,1" 1 -y --from servicer1
lavad tx pairing stake-provider "Ethereum Mainnet" 2000stake "127.0.0.1:2222,grpc,1" 1 -y --from servicer2
lavad tx pairing stake-provider "Ethereum Mainnet" 2050stake "127.0.0.1:2223,grpc,1" 1 -y --from servicer3
lavad tx pairing stake-provider "Ethereum Mainnet" 2020stake "127.0.0.1:2224,grpc,1" 1 -y --from servicer4
lavad tx pairing stake-provider "Ethereum Mainnet" 2030stake "127.0.0.1:2225,grpc,1" 1 -y --from servicer5

#Terra providers
lavad tx pairing stake-provider "Terra Columbus-5 mainnet" 2010stake "127.0.0.1:2241,grpc,1 127.0.0.1:2231,rest,1" 1 -y --from servicer1
lavad tx pairing stake-provider "Terra Columbus-5 mainnet" 2000stake "127.0.0.1:2242,grpc,1 127.0.0.1:2232,rest,1" 1 -y --from servicer2
lavad tx pairing stake-provider "Terra Columbus-5 mainnet" 2050stake "127.0.0.1:2243,grpc,1 127.0.0.1:2233,rest,1" 1 -y --from servicer3
lavad tx pairing stake-provider "Terra Columbus-5 mainnet" 2020stake "127.0.0.1:2244,grpc,1 127.0.0.1:2234,rest,1" 1 -y --from servicer4
lavad tx pairing stake-provider "Terra Columbus-5 mainnet" 2030stake "127.0.0.1:2245,grpc,1 127.0.0.1:2235,rest,1" 1 -y --from servicer5

lavad tx pairing stake-client "Ethereum Mainnet" 200000stake 1 -y --from user1
lavad tx pairing stake-client "Terra Columbus-5 mainnet" 200000stake 1 -y --from user2

echo "---------------Queries------------------"
lavad query pairing providers "Ethereum Mainnet"
lavad query pairing clients "Ethereum Mainnet"
