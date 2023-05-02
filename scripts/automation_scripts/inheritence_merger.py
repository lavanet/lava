import json
import os

## Checks if a spec inherits another spec (recursively) and creates an array that includes all supported rest api calls.
# Edit spec file name that we want to fetch all rest apis including inheritence
spec_file_name = "spec_add_lava.json"

## Constants (Do not edit)
specs_dir = os.getcwd() + "/cookbook/specs/"
spec_current_file_path = specs_dir + spec_file_name
rest_api_list = []

def get_inherited_rest_apis(importName):
    """
    Given a spec "imports" name, this function returns the matching parent spec
    as a Python object. It looks for the spec file recursively in the
    '/specs' directory.
    """

    for filename in os.listdir(specs_dir):
        spec_path = os.path.join(specs_dir, filename)
        if os.path.isfile(spec_path) and spec_path.endswith(".json"):
            with open(spec_path, "r") as f:
                spec_data = f.read()
                data = json.loads(spec_data)
                index = data['proposal']['specs'][0]['index']
                if index == importName:
                    # We have found the parent spec, check if it also has index
                    if 'imports' in  data.get('proposal', {}).get('specs', [{}])[0]:
                        imports = data['proposal']['specs'][0]['imports']
                        if len(imports) > 0:
                            for inheritedSpec in imports:
                                get_inherited_rest_apis(inheritedSpec)
                    else: 
                        print("REACHED ROOT SPEC!: ", index)

                    all_rest_apis = data['proposal']['specs'][0]['apis']
                    for rest_api in all_rest_apis:
                        if rest_api['api_interfaces'][0]['interface'] == 'rest':
                            rest_api = rest_api['name'].split("\"",1)[0]
                            rest_api_list.append(rest_api)
                    print("DONE: ", index)
                    break

    return rest_api_list
                        
# Main
# api_list = get_inherited_rest_apis("LAV1")
