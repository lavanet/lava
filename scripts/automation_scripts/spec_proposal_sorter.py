import glob
import json
import os 
import shutil

all_specs = glob.glob(os.path.join(os.getcwd(), "cookbook/*.json"))
imported = set()
exported = {}

def findMyLevel(import_spec,spec_path):
    highestLevel = 0
    for imported in import_spec:
        for keys in exported:
            for folders in exported[keys]:
                if folders["value"] == spec_path:
                    return keys
                if folders["index"] == imported:
                    if keys > highestLevel:
                        highestLevel = keys
    return highestLevel+1

def addToExported(key,value,index):
    imported.add(index)
    # check if we have this spec already.
    for keys in exported:
        for folders in exported[keys]:
            if folders["value"] == value:
                return 
    if key in exported:
        exported[key].append({"value":value,"index":index})
    else:
        exported[key] = [{"value":value,"index":index}]

def importSpecificSpec(spec_name,highest_level=0):
    for spec in all_specs:
        with open(spec,"r") as fr:
            data = json.load(fr)
        if "proposal" in data:
            if "specs" in data["proposal"]:
                for specInstance in data["proposal"]["specs"]: # spec is an array of specs
                    if "index" in specInstance:
                        if specInstance["index"] in imported:
                            continue
                        if specInstance["index"] == spec_name:
                            if "imports" in specInstance:
                                for import_spec in specInstance["imports"]:
                                    if import_spec not in imported:
                                        level = importSpecificSpec(import_spec,highest_level)
                                        if level > highest_level:
                                            highest_level = level
                                    else:
                                        level = findMyLevel(import_spec,spec)
                                        if level > highest_level:
                                            highest_level = level
                            
                            addToExported(highest_level, spec, spec_name)
                            return highest_level+1
    raise Exception("Shouldn't Reach Here")



for spec in all_specs:
    with open(spec,"r") as fr:
        data = json.load(fr)
    if "proposal" in data:
        if "specs" in data["proposal"]:
            for specInstance in data["proposal"]["specs"]: # spec is an array of specs
                if "index" in specInstance:
                    if specInstance["index"] in imported:
                        continue
                if "imports" in specInstance:
                    for import_spec in specInstance["imports"]:
                        if import_spec not in imported:
                            level = importSpecificSpec(import_spec)
                    myLevel = findMyLevel(specInstance["imports"],spec)
                    addToExported(myLevel, spec, specInstance["index"])
                else:
                    addToExported(0, spec, specInstance["index"])
                    # we dont have any imports we are level 0

# with open("Test.json","w") as fw:
#     data = json.dump(exported,fw,indent=4)

if os.path.exists("SortedSpecs"):
    shutil.rmtree("SortedSpecs")
os.mkdir("SortedSpecs")

pushedExtra = 0

for key in exported.keys():
    
    # Iterate over the list of dictionaries for the key
    for item in exported[key]:
        pathSorted = os.path.join("SortedSpecs",str(int(key)+pushedExtra))
        if not os.path.exists(pathSorted):
            os.mkdir(pathSorted)
        # Get the file path and index from the dictionary
        file_path = item["value"]
        index = item["index"]

        # Copy the file to the destination folder
        shutil.copy2(file_path, pathSorted)

        if os.path.exists(pathSorted):
            number_of_files = len([f for f in os.listdir(pathSorted) if os.path.isfile(os.path.join(pathSorted, f))])
            if number_of_files > 5 and number_of_files != len(exported[key]):
                pushedExtra+=1

