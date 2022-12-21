import glob
import json
all_specs = glob.glob("/home/user/go/src/lava/cookbook/*.json")

for spec in all_specs:
    with open(spec,"r") as fr:
        data = json.load(fr)
    with open(spec,"w") as fw:
        data = json.dump(data,fw,indent=4)
    