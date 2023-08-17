#!/bin/bash
# make install-all
killall -9 lavad

# Function to check if a command is available
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Flag to track if jq installation is successful
jq_installed=false

# Check if jq is not installed
if ! command_exists jq; then
    # Try to install jq using apt (for Debian/Ubuntu)
    if command_exists apt; then
        echo "Installing jq using apt..."
        sudo apt update
        sudo apt install -y jq
        if command_exists jq; then
            echo "jq has been successfully installed."
            jq_installed=true
        fi
    fi

    # Try to install jq using brew (for macOS)
    if ! $jq_installed && command_exists brew; then
        echo "Installing jq using brew..."
        brew install jq
        if command_exists jq; then
            echo "jq has been successfully installed."
            jq_installed=true
        fi
    fi
else
    jq_installed=true
fi

# if jq is still not installed, exit
if ! $jq_installed; then
    echo "Unable to install jq using apt or brew. Please install jq manually."
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

# Check if the OS is macOS or Linux and apply the correct sed command
if [ "$os_name" = "Darwin" ]; then
    # For macOS
    sed -i '' \
    -e 's/timeout_propose = .*/timeout_propose = "1s"/' \
    -e 's/timeout_propose_delta = .*/timeout_propose_delta = "500ms"/' \
    -e 's/timeout_prevote = .*/timeout_prevote = "1s"/' \
    -e 's/timeout_prevote_delta = .*/timeout_prevote_delta = "500ms"/' \
    -e 's/timeout_precommit = .*/timeout_precommit = "500ms"/' \
    -e 's/timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' \
    -e 's/timeout_commit = .*/timeout_commit = "1s"/' \
    -e 's/skip_timeout_commit = .*/skip_timeout_commit = false/' "$path$config"

elif [ "$os_name" = "Linux" ]; then
    # For Linux
    sed -i \
    -e 's/timeout_propose = .*/timeout_propose = "1s"/' \
    -e 's/timeout_propose_delta = .*/timeout_propose_delta = "500ms"/' \
    -e 's/timeout_prevote = .*/timeout_prevote = "1s"/' \
    -e 's/timeout_prevote_delta = .*/timeout_prevote_delta = "500ms"/' \
    -e 's/timeout_precommit = .*/timeout_precommit = "500ms"/' \
    -e 's/timeout_precommit_delta = .*/timeout_precommit_delta = "1s"/' \
    -e 's/timeout_commit = .*/timeout_commit = "1s"/' \
    -e 's/skip_timeout_commit = .*/skip_timeout_commit = false/' "$path$config"

else
    echo "Unsupported OS: $os_name"
    exit 1
fi


# Edit app.toml file
os_name=$(uname)

# Check if the OS is macOS or Linux and apply the correct sed command
if [ "$os_name" = "Darwin" ]; then
    # For macOS
    sed -i '' -e "s/enable = .*/enable = true/" "$path$app"
elif [ "$os_name" = "Linux" ]; then
    # For Linux
    sed -i -e "s/enable = .*/enable = true/" "$path$app"
else
    echo "Unsupported OS: $os_name"
    exit 1
fi
# Add users
users=("alice" "bob" "user1" "user2" "user3" "user4" "servicer1" "servicer2" "servicer3" "servicer4" "servicer5" "servicer6" "servicer7" "servicer8" "servicer9" "servicer10")

for user in "${users[@]}"; do
    lavad keys add "$user"
    lavad add-genesis-account "$user" 50000000000000ulava
done

lavad gentx alice 100000000000ulava --chain-id lava
lavad collect-gentxs
lavad start --pruning=nothing
