import re

files_to_fix = ["./src/grpc_web_services/lavanet/lava/pairing/relay_pb.js","./src/grpc_web_services/lavanet/lava/pairing/badges_pb.js"]

def camel_to_snake(camel_str):
    snake_str = re.sub('([a-z0-9])([A-Z])', r'\1_\2', camel_str)
    return snake_str.lower()

newNames = {}

for file in files_to_fix:
    with open(file, "r") as file_to_read:
        data = file_to_read.read()
    data_split = data.split("var f, obj = {")[1:]
    for d in data_split: 
        names = d.split("}")[0].split(":")
        for name in names:
            if name.strip() == "":
                continue
            name = name.rsplit(" ",1)[-1]
            newname = camel_to_snake(name)
            print(f"{name} => {newname}")
            newNames[name] = newname

    for n in newNames.keys():
        data = data.replace(n, newNames[n])

    with open(file, "w+") as fwrite:
        fwrite.write(data)

