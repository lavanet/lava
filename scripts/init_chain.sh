#!/bin/bash

rm -rf ~/.lava
lavad init validator

lavad config keyring-backend test

# set genesis params
python3 $(dirname "$0")/genesis/prepare_genesis.py

# Define an array of users
users=("alice" "bob" "user1" "user2" "user3" "user4" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10")

# Loop through the array and call the command for each user
for user in "${users[@]}"; do
    lavad keys add "$user"
    lavad add-genesis-account "$user" 50000000000000ulava
done

lavad gentx alice 100000000000ulava --chain-id lava 
lavad collect-gentxs