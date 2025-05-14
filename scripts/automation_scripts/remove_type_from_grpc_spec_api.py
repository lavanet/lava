import json

input_files = [
    "/home/user/go/src/lava/cookbook/specs/cosmoshub.json",
    "/home/user/go/src/lava/cookbook/specs/juno.json",
    "/home/user/go/src/lava/cookbook/specs/lava.json",
    "/home/user/go/src/lava/cookbook/specs/osmosis.json",
]

for f in input_files:
    with open(f,"r") as fr:
        data = json.load(fr)
    for spec in data['proposal']['specs']:
        for api in spec["apis"]:
            for apiInterface in api["api_interfaces"]:
                if apiInterface["interface"] == "grpc":
                    apiInterface["type"] = ""

    with open(f,"w") as fr:
        json.dump(data,fr,indent=4)