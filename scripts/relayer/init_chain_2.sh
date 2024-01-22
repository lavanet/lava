#!/bin/bash
# make install-all
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh

# Check if jq is not installed
if ! command_exists jq; then
    echo "jq not found. Please install jq using the init_install.sh script or manually."
    exit 1
fi

home=~/.lava2
chainID="lava-local-2"
rm -rf $home
lavad init validator --chain-id $chainID --home $home
lavad config broadcast-mode sync --home $home
lavad config keyring-backend test --home $home
lavad config node "tcp://localhost:36657" --home $home


# Specify the file path, field to edit, and new value
path="$home/config/"
genesis='genesis.json'
config='config.toml'
app='app.toml'

# Edit genesis file
if [ "$1" == "debug" ]; then
    # Edit genesis file with additional line
    data=$(cat "$path$genesis" \
        | jq '.app_state.gov.params.min_deposit[0].denom = "ulava"' \
        | jq '.app_state.gov.params.min_deposit[0].amount = "100"' \
        | jq '.app_state.gov.params.voting_period = "4s"' \
        | jq '.app_state.gov.params.expedited_voting_period = "3s"' \
        | jq '.app_state.gov.params.expedited_min_deposit[0].denom = "ulava"' \
        | jq '.app_state.gov.params.expedited_min_deposit[0].amount = "200"' \
        | jq '.app_state.gov.params.expedited_threshold = "0.67"' \
        | jq '.app_state.mint.params.mint_denom = "ulava"' \
        | jq '.app_state.staking.params.bond_denom = "ulava"' \
        | jq '.app_state.crisis.constant_fee.denom = "ulava"' \
        | jq '.app_state.epochstorage.params.epochsToSave = "5"' \
        | jq '.app_state.epochstorage.params.epochBlocks = "4"' \
    )
else
    # Edit genesis file without the additional line
    data=$(cat "$path$genesis" \
        | jq '.app_state.gov.params.min_deposit[0].denom = "ulava"' \
        | jq '.app_state.gov.params.min_deposit[0].amount = "100"' \
        | jq '.app_state.gov.params.voting_period = "4s"' \
        | jq '.app_state.gov.params.expedited_voting_period = "3s"' \
        | jq '.app_state.gov.params.expedited_min_deposit[0].denom = "ulava"' \
        | jq '.app_state.gov.params.expedited_min_deposit[0].amount = "200"' \
        | jq '.app_state.gov.params.expedited_threshold = "0.67"' \
        | jq '.app_state.mint.params.mint_denom = "ulava"' \
        | jq '.app_state.mint.params.mint_denom = "ulava"' \
        | jq '.app_state.staking.params.bond_denom = "ulava"' \
        | jq '.app_state.crisis.constant_fee.denom = "ulava"' \
        | jq '.app_state.downtime.params.downtime_duration = "6s"' \
        | jq '.app_state.downtime.params.epoch_duration = "8s"' \
    )
fi

echo -n "$data" > "$path$genesis"

echo "using genesis file"
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
sed $SED_INLINE -e "/Enable defines if the Rosetta API server should be enabled.*/{n;s/enable = .*/enable = false/}" "$path$app"

# change ports
sed $SED_INLINE \
-e 's/tcp:\/\/0\.0\.0\.0:26656/tcp:\/\/0.0.0.0:36656/' \
-e 's/tcp:\/\/127\.0\.0\.1:26658/tcp:\/\/127.0.0.1:36658/' \
-e 's/tcp:\/\/127\.0\.0\.1:26657/tcp:\/\/127.0.0.1:36657/' \
-e 's/tcp:\/\/127\.0\.0\.1:26656/tcp:\/\/127.0.0.1:36656/' "$path$config"

# Edit app.toml file
sed $SED_INLINE \
-e 's/tcp:\/\/localhost:1317/tcp:\/\/localhost:2317/' \
-e 's/localhost:9090/localhost:8090/' \
-e 's/":7070"/":7070"/' \
-e 's/localhost:9091/localhost:8091/' "$path$app"

# Add users
users=("alice" "bob" "user1" "user2" "user3" "user4" "user5" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10")

for user in "${users[@]}"; do
    lavad keys add "$user" --home $home
    lavad add-genesis-account "$user" 50000000000000ulava --home $home
done

# add validators_allocation_pool for validators block rewards
# its total balance is 3% from the total tokens amount: 10^9 * 10^6 ulava
lavad add-genesis-account validators_rewards_allocation_pool 30000000000000ulava --module-account --home $home
lavad add-genesis-account providers_rewards_allocation_pool 30000000000000ulava --module-account --home $home
lavad gentx alice 10000000000000ulava --chain-id $chainID --home $home
lavad collect-gentxs --home $home
lavad start --pruning=nothing  --home $home