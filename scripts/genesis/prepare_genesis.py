#!/usr/bin/env python3

import json
import toml

# Specify the file path, field to edit, and new value
path = '/home/user/.lava/config/'
genesis = 'genesis.json'
config = 'config.toml'

# Load the JSON file
with open(path + genesis, 'r') as file:
    data = json.load(file)

data["app_state"]["gov"]["deposit_params"]["min_deposit"][0]["denom"] = "ulava"
data["app_state"]["gov"]["deposit_params"]["min_deposit"][0]["amount"] = "10000000"
data["app_state"]["gov"]["voting_params"]["voting_period"] = "3s"
data["app_state"]["mint"]["params"]["mint_denom"] = "ulava"
data["app_state"]["staking"]["params"]["bond_denom"] = "ulava"
data["app_state"]["crisis"]["constant_fee"]["denom"] = "ulava"

# Save the changes back to the JSON file
with open(path + genesis, 'w') as file:
    json.dump(data, file, indent=4)



# Step 1: Read the contents of the .toml file
with open(path + config, 'r') as file:
    data = toml.load(file)

# Step 2: Modify the content as required
data["consensus"]["timeout_propose"] = "1s"
data["consensus"]["timeout_propose_delta"] = "500ms"
data["consensus"]["timeout_prevote"] = "1s"
data["consensus"]["timeout_prevote_delta"] = "500ms"
data["consensus"]["timeout_propose"] = "1s"
data["consensus"]["timeout_precommit"] = "500ms"
data["consensus"]["timeout_precommit_delta"] = "1s"
data["consensus"]["timeout_commit"] = "1s"

# Step 3: Write the updated content back to the .toml file
with open(path + config, 'w') as file:
    toml.dump(data, file)