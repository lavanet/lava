import json, os

path = os.path.dirname(os.path.realpath(__file__)) + "/../../cookbook"

input_files = [
    f"{path}/spec_add_cosmoshub.json",
    f"{path}/spec_add_juno.json",
    f"{path}/spec_add_lava.json",
    f"{path}/spec_add_osmosis.json",
]

for f in input_files:
    with open(f,"r") as fr:
        data = json.load(fr)
    for spec in data['proposal']['specs']:
        for api in spec["apis"]:
            for apiInterface in api["api_interfaces"]:
                if apiInterface["interface"] == "grpc":
                    api["enabled"] = True

    with open(f,"w") as fr:
        json.dump(data,fr,indent=4)