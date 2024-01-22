#!/bin/bash
# make install-all
killall -9 rly
rm -rf ~/.relayer/
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

home2=~/.lava2

rly config init

rly chains add -f $__dir/lav1.json lava-local-1
rly chains add -f $__dir/lav2.json lava-local-2

rly keys add lava-local-1 rly1
rly keys add lava-local-2 rly2

lavad tx bank send alice $(rly keys show lava-local-1 rly1) 10000000ulava -y --from alice
lavad tx bank send alice $(rly keys show lava-local-2 rly2) 10000000ulava -y --from alice --home $home2

rly keys use lava-local-1 rly1
rly keys use lava-local-2 rly2

rly paths new lava-local-1 lava-local-2 demo-path
rly tx link demo-path -d -t 3s
