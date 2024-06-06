#!/bin/bash 

__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

rm data.txt
touch data.txt
while true
do
    echo $(lavad q pairing debug-query $(lavad keys show -a servicer1) ETH1) >> data.txt
    wait_next_block
done