import requests
from datetime import datetime

def fetch_block_data(block_number):
    # URL that returns a JSON response
    url = 'https://lava.rest.lava.build/cosmos/base/tendermint/v1beta1/blocks/' + block_number

    # Fetch the data from the URL
    response = requests.get(url)

    # Parse the JSON data
    json_data = response.json()
    block_header = json_data.get("sdk_block").get("header")

    height = block_header.get("height")

    dt = datetime.strptime(block_header.get("time")[:26] + "Z", "%Y-%m-%dT%H:%M:%S.%fZ")
    time = dt.timestamp()

    proposer = block_header.get("proposer_address")

    return [int(height), float(time), proposer]

stats_before = {}
stats_current = {}
current = fetch_block_data("latest")
for i in range(10000):
    print(f"Progress: {i}", end='\r')
    before = fetch_block_data(str(current[0]-1))
    
    if current[2] not in stats_current:
        stats_current[current[2]] = {"good": 0, "bad": 0}

    if before[2] not in stats_before:
        stats_before[before[2]] = {"good": 0, "bad": 0}

    if current[1] - before[1] < 10:
        stats_current[current[2]]["bad"] += 1
        stats_before[before[2]]["bad"] += 1
    else:
        stats_current[current[2]]["good"] += 1
        stats_before[before[2]]["good"] += 1
    
    current = before

print(stats_before)
print("---------------------------------------------------------------------")
# print(stats_current)


