import csv
import json
import sys
import os

'''
##################################################################################################
## Script goal: Update the CU field in all methods specified in the CSV file

## Script usage: python3 updateSpecsCu <target_folder> (folder should include specs and CSV files)

## CSV file expected structure:
Selector,Method name,Value,Multi10,spec,
Ethereum,eth_accounts,6,60,"ETH1 (ethereum mainnet), GTH1 (ethereum testnet goerli)",
Polygon,debug_getbadblocks,5,50,"POLYGON1 (polygon mainnet), POLYGON1T (polygon testnet)",
Arbitrum,eth_accounts,1,10,ARB1 (arbitrum mainnet),

## Script Assumptions (or in what way you should feed it data so it'll work):
 - spec file must be named with "spec_add" prefix
 - you should edit the correct CSV name in the relevant line (look for UPDATE CSV FILENAME HERE comment)
 - in the CSV's spec column, put the mainnet ID first (for example, ETH1 and then GTH1)
 - after updating the CU, beautify the JSON file

###################################################################################################
'''

def get_spec_ids(line):
    if not line.strip():
        return ["empty line"]
    spec_ids = [spec_id.strip().split()[0] for spec_id in line.split(",")]
    return spec_ids

# Use this func only if spec_add_ethereum.json is present in your folder
def find_non_eth_apis(apis):
    with open("spec_add_ethereum.json", "r") as f_json:
        json_data = json.load(f_json)
        for eth_api in json_data["proposal"]["specs"][0]["apis"]:
            if eth_api["name"] in apis:
                apis.remove(eth_api["name"])
            if eth_api["name"].lower() in apis:
                apis.remove(eth_api["name"].lower())
    
    return apis

# get a folder path from the CLI. Save all the files that start with "spec_add" in a list
folder_path = sys.argv[1]
spec_add_files = []
for filename in os.listdir(folder_path):
    if filename.startswith("spec_add"):
        spec_add_files.append(filename)

csv_data = []

# Open the CSV file and create a reader object. Get the relevant data for the CU update
with open('full_cu.csv', 'r') as f_csv: # <-------------- UPDATE CSV FILENAME HERE
    csv_reader = csv.reader(f_csv)
    header_row = next(csv_reader)

    csv_method_name_index = header_row.index('Method name')
    csv_cu_index = header_row.index('Multi10')
    csv_spec_index = header_row.index('spec')

    for row in csv_reader:
        spec_ids = get_spec_ids(row[csv_spec_index])
        if spec_ids[0] == "empty line":
            continue
        csv_row_data = [spec_ids[0], row[csv_method_name_index], row[csv_cu_index]]
        csv_data.append(csv_row_data)

print("\n#################################")
print("### updateSpecsCu.py started! ###")
print("#################################\n")

for spec_file in spec_add_files:
    csv_apis_of_spec_without_apis = []
    api_changed_counter = 0

    with open(spec_file, 'r') as f_json:
        json_data = json.load(f_json)
        print("# Proccessing " + spec_file)

        # take only mainnet
        json_spec_data = json_data["proposal"]["specs"][0]
        # iterate on the csv data
        for csv_data_row in csv_data:

            # if the spec index is equal, iterate on the spec's apis
            if json_spec_data["index"] == csv_data_row[0]:
                api_found = False
                if "apis" in json_spec_data:
                    for api in json_spec_data["apis"]:
                        # if the method name is equal, update the CU
                        if api["name"] == csv_data_row[1] or api["name"].lower() == csv_data_row[1]:
                            api["compute_units"] = csv_data_row[2]
                            api_found = True
                            api_changed_counter += 1
                            break
                    if not api_found:
                        non_eth_api = find_non_eth_apis([csv_data_row[1]])
                        if non_eth_api:
                            print(" - API from CSV which was not found: " + non_eth_api[0])
                else:
                    csv_apis_of_spec_without_apis.append(csv_data_row[1])

    if csv_apis_of_spec_without_apis:
        print(" - APIs from CSV not changed (spec has no \"apis\" field): " + ', '.join(find_non_eth_apis(csv_apis_of_spec_without_apis)))

    print("Done. Changed "+ str(api_changed_counter) + " APIs!\n")

    # write the updated JSON data back to file
    f_json.close()
    with open(spec_file, 'w') as f_json:
        json.dump(json_data, f_json, indent=4)