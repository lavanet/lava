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
if [ "$1" == "debug" ]; then
    # Edit genesis file with additional line
    data=$(cat "$path$genesis" \
  | jq '.consensus_params.block.max_bytes = "22020096"' \
    | jq '.consensus_params.block.max_gas = "-1"' \
    | jq '.consensus_params.evidence.max_age_num_blocks = "100000"' \
    | jq '.consensus_params.evidence.max_age_duration = "172800000000000"' \
    | jq '.consensus_params.evidence.max_bytes = "1048576"' \
    | jq '.app_state.auth.params.max_memo_characters = "256"' \
    | jq '.app_state.auth.params.tx_sig_limit = "7"' \
    | jq '.app_state.auth.params.tx_size_cost_per_byte = "10"' \
    | jq '.app_state.auth.params.sig_verify_cost_ed25519 = "590"' \
    | jq '.app_state.auth.params.sig_verify_cost_secp256k1 = "1000"' \
    | jq '.app_state.distribution.params.community_tax = "0.020000000000000000"' \
    | jq '.app_state.distribution.params.base_proposer_reward = "0.000000000000000000"' \
    | jq '.app_state.distribution.params.bonus_proposer_reward = "0.000000000000000000"' \
    | jq '.app_state.distribution.params.withdraw_addr_enabled = true' \
    | jq '.app_state.gov.params.max_deposit_period = "172800s"' \
    | jq '.app_state.gov.params.min_deposit[0].denom = "ulava"' \
    | jq '.app_state.gov.params.min_deposit[0].amount = "7000000000"' \
    | jq '.app_state.gov.params.burn_proposal_deposit_prevote = false' \
    | jq '.app_state.gov.params.burn_vote_quorum = false' \
    | jq '.app_state.gov.params.burn_vote_veto = true' \
    | jq '.app_state.gov.params.expedited_min_deposit[0].denom = "ulava"' \
    | jq '.app_state.gov.params.expedited_min_deposit[0].amount = "14000000000"' \
    | jq '.app_state.gov.params.expedited_threshold = "0.667"' \
    | jq '.app_state.gov.params.expedited_voting_period = "3s"' \
    | jq '.app_state.gov.params.min_initial_deposit_ratio = "0.250000000000000000"' \
    | jq '.app_state.gov.params.quorum = "0.334000000000000000"' \
    | jq '.app_state.gov.params.threshold = "0.500000000000000000"' \
    | jq '.app_state.gov.params.veto_threshold = "0.334000000000000000"' \
    | jq '.app_state.gov.params.voting_period = "4s"' \
    | jq '.app_state.ibc.client_genesis.params.allowed_clients = ["07-tendermint"]' \
    | jq '.app_state.ibc.connection_genesis.params.max_expected_time_per_block = "30000000000"' \
    | jq '.app_state.transfer.params.send_enabled = true' \
    | jq '.app_state.transfer.params.receive_enabled = true' \
    | jq '.app_state.slashing.params.downtime_jail_duration = "600s"' \
    | jq '.app_state.slashing.params.min_signed_per_window = "0.500000000000000000"' \
    | jq '.app_state.slashing.params.signed_blocks_window = "50"' \
    | jq '.app_state.slashing.params.slash_fraction_double_sign = "0.000000000000000000"' \
    | jq '.app_state.slashing.params.slash_fraction_downtime = "0.000000000000000000"' \
    | jq '.app_state.staking.params.bond_denom = "ulava"' \
    | jq '.app_state.staking.params.historical_entries = "10000"' \
    | jq '.app_state.staking.params.max_entries = "50"' \
    | jq '.app_state.staking.params.max_validators = "100"' \
    | jq '.app_state.staking.params.min_commission_rate = "0.010000000000000000"' \
    | jq '.app_state.staking.params.unbonding_time = "1814400s"' \
    | jq '.app_state.bank.denom_metadata[0].description = "The native token of the network"' \
    | jq '.app_state.bank.denom_metadata[0].denom_units[0].denom = "ulava"' \
    | jq '.app_state.bank.denom_metadata[0].denom_units[1].denom = "lava"' \
    | jq '.app_state.bank.denom_metadata[0].denom_units[1].exponent = "6"' \
    | jq '.app_state.bank.denom_metadata[0].base = "ulava"' \
    | jq '.app_state.bank.denom_metadata[0].name = "LAVA"' \
    | jq '.app_state.bank.denom_metadata[0].display = "lava"' \
    | jq '.app_state.bank.denom_metadata[0].symbol = "lava"' \
    | jq '.app_state.bank.supply[0].denom = "ulava"' \
    | jq '.app_state.bank.supply[0].amount = "1000000000000000"' \
    | jq '.app_state.crisis.constant_fee.denom = "ulava"' \
    | jq '.app_state.crisis.constant_fee.amount = "50000000000000"' \
    | jq '.app_state.conflict.params.Rewards.clientRewardPercent = "0.000000000000000000"' \
    | jq '.app_state.conflict.params.Rewards.votersRewardPercent = "0.000000000000000000"' \
    | jq '.app_state.conflict.params.Rewards.winnerRewardPercent = "0.000000000000000000"' \
    | jq '.app_state.conflict.params.majorityPercent = "0.999000000000000000"' \
    | jq '.app_state.conflict.params.votePeriod = "2"' \
    | jq '.app_state.conflict.params.voteStartSpan = "3"' \
    | jq '.app_state.downtime.params.downtime_duration = "1s"' \
    | jq '.app_state.downtime.params.epoch_duration = "8s"' \
    | jq '.app_state.epochstorage.params.epochBlocks = "8"' \
    | jq '.app_state.epochstorage.params.epochsToSave = "1"' \
    | jq '.app_state.epochstorage.params.latestParamChange = "0"' \
    | jq '.app_state.epochstorage.params.unstakeHoldBlocks = "6040"' \
    | jq '.app_state.epochstorage.params.unstakeHoldBlocksStatic = "6041"' \
    | jq '.app_state.pairing.params.QoSWeight = "0.500000000000000000"' \
    | jq '.app_state.pairing.params.epochBlocksOverlap = "5"' \
    | jq '.app_state.pairing.params.recommendedEpochNumToCollectPayment = "3"' \
    | jq '.app_state.protocol.params.version.provider_target = "0.33.1"' \
    | jq '.app_state.protocol.params.version.provider_min = "0.33.1"' \
    | jq '.app_state.protocol.params.version.consumer_target = "0.33.1"' \
    | jq '.app_state.protocol.params.version.consumer_min = "0.33.1"' \
    | jq '.app_state.rewards.params.leftover_burn_rate = "1.000000000000000000"' \
    | jq '.app_state.rewards.params.low_factor = "0.5"' \
    | jq '.app_state.rewards.params.max_bonded_target = "0.8"' \
    | jq '.app_state.rewards.params.min_bonded_target = "0.6"' \
    | jq '.app_state.rewards.params.max_reward_boost = "5"' \
    | jq '.app_state.rewards.params.validators_subscription_participation = "0.050000000000000000"' \
    | jq '.app_state.spec.params.maxCU = "10000"' \
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


# Add users
users=("alice" "user1" "servicer1" "servicer2" "servicer3")

for user in "${users[@]}"; do
    lavad keys add "$user"
    lavad add-genesis-account "$user" 199999980000000ulava
done

# add validators_allocation_pool for validators block rewards
# its total balance is 3% from the total tokens amount: 10^9 * 10^6 ulava
lavad add-genesis-account validators_rewards_allocation_pool 34000000ulava --module-account 
lavad add-genesis-account providers_rewards_allocation_pool 66000000ulava --module-account 
lavad gentx alice 10000000000000ulava --chain-id lava
lavad collect-gentxs
lavad start --pruning=nothing
