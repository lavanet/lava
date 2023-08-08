#!/bin/bash
# make install

rm -rf ~/.lava
lavad init validator

lavad config broadcast-mode block
lavad config keyring-backend test

# Specify the file path, field to edit, and new value
path="$HOME/.lava/config/"
genesis='genesis.json'
config='config.toml'
app='app.toml'

# Edit genesis file
data=$(cat "$path$genesis" \
    | jq '.app_state.gov.deposit_params.min_deposit[0].denom = "ulava"' \
    | jq '.app_state.gov.deposit_params.min_deposit[0].amount = "10000000"' \
    | jq '.app_state.gov.voting_params.voting_period = "3s"' \
    | jq '.app_state.mint.params.mint_denom = "ulava"' \
    | jq '.app_state.staking.params.bond_denom = "ulava"' \
    | jq '.app_state.crisis.constant_fee.denom = "ulava"' \
    )

echo -n "$data" > "$path$genesis"

# Edit conflict.toml file
sed -i "$path$config" \
    -e 's/timeout_propose = .*/timeout_propose = "1s"/' \
    -e 's/timeout_propose_delta = .*/timeout_propose_delta = "500ms"/' \
    -e 's/timeout_prevote = .*/timeout_prevote = "1s"/' \
    -e 's/timeout_prevote_delta = .*/timeout_prevote_delta = "500ms"/' \
    -e 's/timeout_precommit = .*/timeout_precommit = "500ms"/' \
    -e 's/timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' \
    -e 's/timeout_commit = .*/timeout_commit = "1s"/' \
    -e 's/skip_timeout_commit = .*/skip_timeout_commit = false/' 

# Edit app.toml file
sed -i '' -e "s/enable = .*/enable = true/" "$path$app"

# Add users
users=("alice" "bob" "user1" "user2" "user3" "user4" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10")

for user in "${users[@]}"; do
    lavad keys add "$user"
    lavad add-genesis-account "$user" 50000000000000ulava
done

lavad gentx alice 100000000000ulava --chain-id lava
lavad collect-gentxs
lavad start