#!/bin/bash
# make install-all
killall -9 lavad
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/useful_commands.sh

# Check if jq is not installed
if ! command_exists jq; then
    echo "jq not found. Please install jq using the init_install.sh script or manually."
    exit 1
fi

rm -rf ~/.lava
lavad init validator --chain-id lava
lavad config broadcast-mode sync
lavad config keyring-backend test

# Specify the file path, field to edit, and new value
path="$HOME/.lava/config/"
genesis='genesis.json'
config='config.toml'
app='app.toml'

# Edit genesis file
data=$(cat "$path$genesis" \
    | jq '.app_state.downtime.params.downtime_duration = "10s"' \
    | jq '.app_state.downtime.params.epoch_duration = "30s"' \
    | jq '.app_state.gov.params.min_deposit[0].denom = "ulava"' \
    | jq '.app_state.gov.params.min_deposit[0].amount = "100"' \
    | jq '.app_state.gov.params.voting_period = "3s"' \
    | jq '.app_state.mint.params.mint_denom = "ulava"' \
    | jq '.app_state.staking.params.bond_denom = "ulava"' \
    | jq '.app_state.crisis.constant_fee.denom = "ulava"' \
    )

echo -n "$data" > "$path$genesis"

echo $(cat "$path$genesis")

# Determine OS
os_name=$(uname)
case "$(uname)" in
  Darwin)
    SED_INLINE="-i ''" ;;
  Linux)
    SED_INLINE="-i" ;;
  *)
    echo "unknown system: $(uname)"
    exit 1 ;;
esac


sed $SED_INLINE \
-e 's/timeout_propose = .*/timeout_propose = "1s"/' \
-e 's/timeout_propose_delta = .*/timeout_propose_delta = "500ms"/' \
-e 's/timeout_prevote = .*/timeout_prevote = "1s"/' \
-e 's/timeout_prevote_delta = .*/timeout_prevote_delta = "500ms"/' \
-e 's/timeout_precommit = .*/timeout_precommit = "500ms"/' \
-e 's/timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' \
-e 's/timeout_commit = .*/timeout_commit = "1s"/' \
-e 's/skip_timeout_commit = .*/skip_timeout_commit = false/' "$path$config"

# Edit app.toml file
sed $SED_INLINE -e "s/enable = .*/enable = true/" "$path$app"

# Add users
users=("alice" "bob" "user1" "user2" "user3" "user4" "user5" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10")

for user in "${users[@]}"; do
    lavad keys add "$user"
    lavad add-genesis-account "$user" 50000000000000ulava
done

lavad gentx alice 100000000000ulava --chain-id lava
lavad collect-gentxs
lavad start --pruning=nothing
