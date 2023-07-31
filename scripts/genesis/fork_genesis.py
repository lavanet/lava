import json

# Specify the file path, field to edit, and new value
path = '/home/user/go/lava/scripts/genesis/'
genesis_org = 'stg_genesis.100723.json'
genesis = 'genesis.json'
genesis_specs = 'genesis_specs.json'

# Load the JSON file
with open(path + genesis_org, 'r') as file:
    data = json.load(file)

data["consensus_params"]["block"]["time_iota_ms"] = '1'
data["app_state"]["gov"]["proposals"] = []

# give providers back their money
for spec in data["app_state"]["spec"]["specList"]:
    for stakeStorage in data["app_state"]["epochstorage"]["stakeStorageList"]:
        if stakeStorage["index"] == spec["index"]:
            for entry in stakeStorage["stakeEntries"]:
                for bankAdd in data["app_state"]["bank"]["balances"]:
                    if bankAdd["address"] == entry["address"]:
                       bankAdd["coins"][0]["amount"] = str(int(bankAdd["coins"][0]["amount"]) + int(entry["stake"]["amount"]))
                       data["app_state"]["bank"]["supply"][0]["amount"] = str(int(data["app_state"]["bank"]["supply"][0]["amount"]) + int(entry["stake"]["amount"]))
                       break

data["app_state"]["epochstorage"]["stakeStorageList"] = []
data["app_state"]["epochstorage"]["params"]["epochBlocks"] = "60"
data["app_state"]["epochstorage"]["params"]["unstakeHoldBlocks"] = "3020"
data["app_state"]["epochstorage"]["params"]["unstakeHoldBlocksStatic"] = "3100"
data["app_state"]["epochstorage"]["params"]["latestParamChange"] = data["initial_height"] # fixate the params on start
data["app_state"]["epochstorage"]["epochDetails"]["earliestStart"] = data["initial_height"]


data["app_state"]["conflict"]["conflictVoteList"] = []
data["app_state"]["pairing"]["epochPaymentsList"] = []
data["app_state"]["pairing"]["providerPaymentStorageList"] = []
data["app_state"]["pairing"]["uniquePaymentStorageClientProviderList"] = []

data["chain_id"] = "lava-testnet-2"


with open(path + genesis_specs, 'r') as file:
    data_specs = json.load(file)

data["app_state"]["spec"]["specList"] = data_specs["app_state"]["spec"]["specList"]
data["app_state"]["spec"]["specCount"] = data_specs["app_state"]["spec"]["specCount"]


# Save the changes back to the JSON file
with open(path + genesis, 'w') as file:
    json.dump(data, file, indent=4)

