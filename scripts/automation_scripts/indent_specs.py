import glob
import json
import os 

all_specs = glob.glob(os.path.join(os.getcwd() + "cookbook/*.json"))

for spec in all_specs:
    with open(spec,"r") as fr:
        data = json.load(fr)
    with open(spec,"w") as fw:
        data = json.dump(data,fw,indent=4)