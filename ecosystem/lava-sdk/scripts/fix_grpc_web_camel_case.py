import re

files_to_fix = ["./src/grpc_web_services/lavanet/lava/pairing/relay_pb.js","./src/grpc_web_services/lavanet/lava/pairing/badges_pb.js"]

print("""
+++++++++++++++++++++++++++++++++++++++++++++++++++
+                                                 +
+           fix_grpc_web_camel_case               +
+       fixing grpc web protobuf camel case       +
+                                                 +
+             types that are fixed                +
+++++++++++++++++++++++++++++++++++++++++++++++++++
""")
      

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
            if newname.endswith("_list"):
                # remove auto appended _list form names since it doesnt work well with protobuf
                newname = newname[:-5]
            print(f"{name} => {newname}")
            newNames[name] = newname

    for n in newNames.keys():
        data = data.replace(n, newNames[n])
    data = data.replace("content_hash: msg.getContentHash_asB64(),","content_hash: msg.getContentHash_asU8(),") # we need it as uint8 not as string when serializing
    with open(file, "w+") as fwrite:
        fwrite.write(data)

print("Finished fix_grpc_web_camel_case.py")