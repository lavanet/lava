#!/bin/bash
# this scripts boots up a relayer between the 2 local chain (using init_chain.sh and init_chain_2.sh)
killall -9 rly
rm -rf ~/.relayer/
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

home2=~/.lava2

rly config init

rly chains add -f $__dir/lav1.json lava
rly chains add -f $__dir/osmosis.json osmosis

rly keys add lava rly1
rly keys add osmosis rly2

osmosisd tx bank send validator1 $(rly keys show osmosis rly2) 10000000stake --fees 500stake -y --from validator1 --keyring-backend=test --home=$HOME/.osmosisd/validator1 --chain-id=testing7
sleep 5
osmosisd tx bank send validator1 $(rly keys show osmosis rly2) 10000000uosmo --fees 500stake -y --from validator1 --keyring-backend=test --home=$HOME/.osmosisd/validator1 --chain-id=testing7
lavad tx bank send yarom $(rly keys show lava rly1) 10000ulava -y --from yarom

rly keys use lava rly1
rly keys use osmosis rly2

osmosisd q bank balances $(rly keys show osmosis rly2)

# rly paths new lava-staging-4 testing7 demo-path
# rly tx link demo-path -d -t 33s
# rly start demo-path

