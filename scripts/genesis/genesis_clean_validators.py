import json

# Specify the file path, field to edit, and new value
path = '/home/user/go/lava/scripts/genesis/'
genesis_org = 'genesis_specs.json'
genesis = 'genesis.json'

distributionModule = "lava@1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8l8newj"
bondedPool = "lava@1fl48vsnmsdzcv85q5d2q4z5ajdha8yu3dr7276"

# Load the JSON file
with open(path + genesis_org, 'r') as file:
    data = json.load(file)

data["app_state"]["epochstorage"]["params"]["epochBlocks"] = "60"
data["app_state"]["epochstorage"]["params"]["unstakeHoldBlocks"] = "3020"
data["app_state"]["epochstorage"]["params"]["unstakeHoldBlocksStatic"] = "3100"
data["app_state"]["epochstorage"]["params"]["latestParamChange"] = data["initial_height"] # fixate the params on start
data["app_state"]["epochstorage"]["epochDetails"]["earliestStart"] = data["initial_height"]

data['app_state']["ibc"]["channel_genesis"]["ack_sequences"] = []
data['app_state']["ibc"]["channel_genesis"]["acknowledgements"] = []
data['app_state']["ibc"]["channel_genesis"]["channels"] = []
data['app_state']["ibc"]["channel_genesis"]["commitments"] = []
data['app_state']["ibc"]["channel_genesis"]["receipts"] = []
data['app_state']["ibc"]["channel_genesis"]["recv_sequences"] = []
data['app_state']["ibc"]["channel_genesis"]["send_sequences"] = []

data['app_state']["ibc"]["client_genesis"]["clients"] = []
data['app_state']["ibc"]["client_genesis"]["clients_consensus"] = []
data['app_state']["ibc"]["client_genesis"]["clients_metadata"] = []

data["app_state"]["gov"]["proposals"] = []
data["validators"] = []
data["app_state"]["staking"]["last_validator_powers"] = "0"
data["app_state"]["staking"]["validators"] = []
data["app_state"]["staking"]["delegations"] = []
data["app_state"]["staking"]["last_validator_powers"] = []

data["app_state"]["distribution"]["delegator_starting_infos"] = []
data["app_state"]["distribution"]["outstanding_rewards"] = []
data["app_state"]["distribution"]["validator_accumulated_commissions"] = []
data["app_state"]["distribution"]["validator_current_rewards"] = []
data["app_state"]["distribution"]["validator_historical_rewards"] = []
data["app_state"]["distribution"]["validator_slash_events"] = []

data["app_state"]["conflict"]["conflictVoteList"] = []
data["app_state"]["pairing"]["epochPaymentsList"] = []
data["app_state"]["pairing"]["providerPaymentStorageList"] = []
data["app_state"]["pairing"]["uniquePaymentStorageClientProviderList"] = []
data["app_state"]["pairing"]["badgeUsedCuList"] = []

data["chain_id"] = "lava"

data["app_state"]["bank"]["supply"] = []

for bankAdd in data["app_state"]["bank"]["balances"]:
        if bankAdd["address"] == distributionModule:
            bankAdd["coins"][0]["amount"] = str(int(float(data["app_state"]["distribution"]["fee_pool"]["community_pool"][0]["amount"])))
            break
        

for bankAdd in data["app_state"]["bank"]["balances"]:
        if bankAdd["address"] == bondedPool:
            bankAdd["coins"] = []
            break

# Save the changes back to the JSON file
with open(path + genesis, 'w') as file:
    json.dump(data, file, indent=4)

