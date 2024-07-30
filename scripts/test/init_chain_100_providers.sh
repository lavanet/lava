#!/bin/bash
# make install-all
killall -9 lavad
__dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source $__dir/../useful_commands.sh

# Check if jq is not installed
if ! command_exists jq; then
    echo "jq not found. Please install jq using the init_install.sh script or manually."
    exit 1
fi

rm -rf ~/.lava
chainID="lava"
lavad init validator --chain-id $chainID
lavad config broadcast-mode sync
lavad config keyring-backend test

# Specify the file path, field to edit, and new value
path="$HOME/.lava/config/"
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
        | jq '.app_state.distribution.params.community_tax = "0"' \
        | jq '.app_state.rewards.params.validators_subscription_participation = "0"' \
        | jq '.app_state.downtime.params.downtime_duration = "1s"' \
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
        | jq '.app_state.downtime.params.epoch_duration = "10s"' \
        | jq '.app_state.epochstorage.params.epochsToSave = "8"' \
        | jq '.app_state.epochstorage.params.epochBlocks = "20"' \
        | jq '.app_state.pairing.params.recommendedEpochNumToCollectPayment = "2"' \
    )
fi

echo -n "$data" > "$path$genesis"

echo "using genesis file"
echo "$(cat "$path$genesis")"

# Determine OS
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

# Add users
users=("alice" "bob" "user1" "user2" "user3" "user4" "user5" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10" "useroren1" "useroren2" "useroren3" "useroren4" "useroren5" "useroren6" "useroren7" "useroren8" "useroren9" "useroren10" "useroren11" "useroren12" "useroren13" "useroren14" "useroren15" "useroren16" "useroren17" "useroren18" "useroren19" "useroren20" "useroren21" "useroren22" "useroren23" "useroren24" "useroren25" "useroren26" "useroren27" "useroren28" "useroren29" "useroren30" "useroren31" "useroren32" "useroren33" "useroren34" "useroren35" "useroren36" "useroren37" "useroren38" "useroren39" "useroren40" "useroren41" "useroren42" "useroren43" "useroren44" "useroren45" "useroren46" "useroren47" "useroren48" "useroren49" "useroren50" "useroren51" "useroren52" "useroren53" "useroren54" "useroren55" "useroren56" "useroren57" "useroren58" "useroren59" "useroren60" "useroren61" "useroren62" "useroren63" "useroren64" "useroren65" "useroren66" "useroren67" "useroren68" "useroren69" "useroren70" "useroren71" "useroren72" "useroren73" "useroren74" "useroren75" "useroren76" "useroren77" "useroren78" "useroren79" "useroren80" "useroren81" "useroren82" "useroren83" "useroren84" "useroren85" "useroren86" "useroren87" "useroren88" "useroren89" "useroren90" "useroren91" "useroren92" "useroren93" "useroren94" "useroren95" "useroren96" "useroren97" "useroren98" "useroren99" "useroren100")

for user in "${users[@]}"; do
    lavad keys add "$user" --keyring-backend test
    lavad add-genesis-account "$user" 50000000000000ulava --keyring-backend test
done

# add validators_allocation_pool for validators block rewards
# its total balance is 3% from the total tokens amount: 10^9 * 10^6 ulava
lavad add-genesis-account validators_rewards_allocation_pool 30000000000000ulava --module-account
if [ "$1" == "debug" ]; then
    lavad add-genesis-account providers_rewards_allocation_pool 0ulava --module-account
else
    lavad add-genesis-account providers_rewards_allocation_pool 30000000000000ulava --module-account 
fi
lavad add-genesis-account iprpc_pool 0ulava --module-account
lavad gentx alice 10000000000000ulava --chain-id $chainID --keyring-backend test
lavad collect-gentxs
lavad start --pruning=nothing