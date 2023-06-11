#!/bin/bash 

set -e

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

GASPRICE="0.000000001ulava"

# Specs proposal
lavad tx gov submit-proposal spec-add ./cookbook/specs/spec_add_ethereum.json,./cookbook/specs/spec_add_ibc.json,./cookbook/specs/spec_add_cosmossdk.json,./cookbook/specs/spec_add_lava.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 1 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

# Plans proposal
lavad tx gov submit-proposal plans-add ./cookbook/plans/default.json,./cookbook/plans/temporary-add.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 2 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep 4

# Plan removal (of one)
lavad tx gov submit-proposal plans-del ./cookbook/plans/temporary-del.json -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx gov vote 3 yes -y --from alice --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

sleep_until_next_epoch

STAKE="500000000000ulava"
# Ethereum providers
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2221,jsonrpc,1" 1 -y --from servicer1 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2222,jsonrpc,1" 1 -y --from servicer2 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2223,jsonrpc,1" 1 -y --from servicer3 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2224,jsonrpc,1" 1 -y --from servicer4 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "ETH1" $STAKE "127.0.0.1:2225,jsonrpc,1" 1 -y --from servicer5 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# Lava tendermint/rest providers
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2261,tendermintrpc,1 127.0.0.1:2271,rest,1 127.0.0.1:2281,grpc,1" 1 -y --from servicer6 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2262,tendermintrpc,1 127.0.0.1:2272,rest,1 127.0.0.1:2282,grpc,1" 1 -y --from servicer7 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2263,tendermintrpc,1 127.0.0.1:2273,rest,1 127.0.0.1:2283,grpc,1" 1 -y --from servicer8 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2264,tendermintrpc,1 127.0.0.1:2274,rest,1 127.0.0.1:2284,grpc,1" 1 -y --from servicer9 --provider-moniker "dummyMoniker" --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-provider "LAV1" $STAKE "127.0.0.1:2265,tendermintrpc,1 127.0.0.1:2275,rest,1 127.0.0.1:2285,grpc,1" 1 -y --from servicer10 --provider-moniker "dummyMoniker"  --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# staked clients
# (actually, only user1 is used in testutils/e2e/e2e.go, but having same count
# in both chains simplifies the e2e logic that expects same amount of staked
# clients in all tested chains)
lavad tx pairing stake-client "ETH1" $STAKE 1 -y --from user1 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
lavad tx pairing stake-client "LAV1" $STAKE 1 -y --from user2 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

# subscriptions and projects
lavad tx subscription buy "DefaultPlan" -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE

user3addr=$(lavad keys show user3 -a)

lavad tx subscription add-project "myproject" ./cookbook/projects/example_policy.yml -y --from user3 --gas-adjustment "1.5" --gas "auto" --gas-prices $GASPRICE
sleep_until_next_epoch

lavad q subscription list-projects user3addr

lavad tx project add-keys -y "$user3addr-myproject" --from user3 cookbook/projects/example_project_keys.yml --gas-prices=$GASPRICE
sleep_until_next_epoch

lavad tx project del-keys -y "$user3addr-myproject" --from user3 cookbook/projects/example_project_keys.yml --gas-prices=$GASPRICE
sleep_until_next_epoch

# we need to wait for the next epoch for the stake to take action.
sleep_until_next_epoch
