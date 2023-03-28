import subprocess
import os

# Lava:
# grpc_server = "public-rpc.lavanet.xyz:9090"
# spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_lava.json"
# chain = "Lava"

# Osmosis:
# grpc_server = "prod-pnet-osmosisnode-1.lavapro.xyz:9090"
# spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_osmosis.json" 
# chain = "Osmosis"

# Cosmos
# grpc_server = "gaia-node-1.lavapro.xyz:9090"
# spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_cosmoshub.json"
# chain = "Cosmos"

# JUNO
grpc_server = "juno-node-1.lavapro.xyz:9090"
spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_juno.json"
chain = "Juno"


result_dir = "/home/user/go/src/lava/scripts/automation_scripts/automation_results/grpcClientProtobufs"

os.makedirs(result_dir,exist_ok=True,mode = 0o777)

with open(spec_current_file_path, 'r') as spec:
  spec_data = spec.read()
grpc_amount = spec_data.count('"interface": "grpc"')

if spec_data.count("chainid") > 2:
  spec_data = spec_data.rsplit('"chainid"',1)[0]
spec_data = spec_data.split('"apis":',1)[-1]
spec_data = spec_data.split('"name":')

grpc_api = []
for api in spec_data:
  if '"interface": "grpc"' not in api:
    continue
  if "get" in api and "post" in api:
    grpc_amount-=1 # decrease amount when have two types
  grpc_api.append(api)


if len(grpc_api) != grpc_amount:
  raise ValueError("len(grpc_api) != grpc_amount",len(grpc_api), grpc_amount)


def getSubDirectoryParentName(method: str):
  splitted = method.split(".")
  directoryName = ""
  ParentDirectoryName = ""
  for x in splitted:
    if splitted.index(x) == 0:
      ParentDirectoryName = x[0].upper() + x[1:]
    if x == "Query" or x == "Service":
      break
    directoryName += x[0].upper() + x[1:]
  return ParentDirectoryName,directoryName

def getFunctionName(method: str):
  return method.rsplit(".",1)[-1]

def getType(method: str):
  type_ret = ""
  if ".Query." in method: 
    type_ret = "Query"
  elif ".Service." in method: 
    type_ret = "Service"
  else:
    raise ValueError("Didnt find Query/Service type in", method)
  return type_ret

info = {}
print("loading all descriptors, this might take few mins")
for x in grpc_api:
  method_name = x.split(",",1)[0].replace("\"","").strip()
  parent_dir, child_dir = getSubDirectoryParentName(method_name)
  func_name = getFunctionName(method_name)
  service_type = getType(method_name)
  descriptor = str(subprocess.check_output(f"grpcurl -plaintext {grpc_server} describe {method_name}",shell=True))[2:-1]
  request = descriptor.split(func_name,1)[-1].strip().split("returns")[0].rsplit(".",1)[-1].split(" ")[0].strip()
  if not request[0].isupper(): 
    raise ValueError("reuqest parsing issue request:", request,"\n descriptor:\n", descriptor)
  
  response = descriptor.split("returns",1)[-1].split("{",1)[0].rsplit(".",1)[-1].split(" ",1)[0].strip()
  if not response[0].isupper(): 
    raise ValueError("reuqest parsing issue response:", response,"\n descriptor:\n", descriptor)

  info[method_name] = {
    "parent_dir": parent_dir, 
    "child_dir": child_dir,
    "func_name": func_name,
    "type": service_type,
    "request": request,
    "response": response,
  }

scaffold_implemented_service = "// this line is used by grpc_scaffolder #implemented###UNIQUENAME###"
scaffold_methods = "// this line is used by grpc_scaffolder #Methods"
scaffold_method = "// this line is used by grpc_scaffolder #Method"
scaffold_register = "// this line is used by grpc_scaffolder #Register"
scaffold_registeration = "// this line is used by grpc_scaffolder #Registration"

FILE_TEMPLATE = f"""
package ###PACKAGE_NAME###_thirdparty

import (
	"context"
	"encoding/json"

	// add protobuf here as pb_pkg
	"github.com/lavanet/lava/utils"
	"google.golang.org/protobuf/proto"
)

type implemented###UNIQUENAME### struct {"{"}
	pb_pkg.Unimplemented###SERVICE_OR_QUERY###Server
	cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)
{"}"}

{scaffold_implemented_service}

{scaffold_methods}
"""

METHOD_TEMPLATE = f"""

func (is *implemented###UNIQUENAME###) ###METHOD_NAME###(ctx context.Context, req *pb_pkg.###REQUEST###) (*pb_pkg.###RESPONSE###, error) {"{"}
	reqMarshaled, err := json.Marshal(req)
	if err != nil {"{"}
		return nil, utils.LavaFormatError("Failed to proto.Marshal(req)", err)
	{"}"}
	res, err := is.cb(ctx, "###WHOLE_SERVICE###", reqMarshaled)
	if err != nil {"{"}
		return nil, utils.LavaFormatError("Failed to SendRelay cb", err)
	{"}"}
	result := &pb_pkg.###RESPONSE###{"{}"}
	err = proto.Unmarshal(res, result)
	if err != nil {"{"}
		return nil, utils.LavaFormatError("Failed to proto.Unmarshal", err)
	{"}"}
	return result, nil
{"}"}
{scaffold_method}

"""

childrens = {}
for i in info:
  parent_name:str = info[i]['parent_dir']
  parent_path = os.path.join(result_dir,parent_name)
  child_name:str = info[i]['child_dir']
  child_path = os.path.join(parent_path,child_name+".go")
  childrens[i] = {"struct_name":child_name,
    "package_name": parent_name.lower(),
    "child_name": child_name,
    "server_type": info[i]['type'],
    "parent_name":parent_name,  
    "parent_path": parent_path
  }

  if not os.path.exists(parent_path):
    os.makedirs(parent_path,exist_ok=False,mode = 0o777)
  if os.path.exists(child_path): # file exists, extend it 
    with open(child_path, 'r') as child_file:
      child_data = child_file.read()
    if info[i]["func_name"] in child_data:
      continue # continue if we have this function already (from past might be or duplicates)
    child_data = child_data.replace(scaffold_methods, 
      METHOD_TEMPLATE.replace("###UNIQUENAME###",child_name).replace("###METHOD_NAME###",info[i]["func_name"]).
      replace("###REQUEST###",info[i]["request"]).
      replace("###RESPONSE###",info[i]["response"]).replace("###WHOLE_SERVICE###",i) + 
      scaffold_methods)
    with open(child_path, 'w+') as ex_file:
      ex_file.write(child_data)

  else: # file is new create from scratch
    with open(child_path, 'w+') as new_file:
      tmp_template = FILE_TEMPLATE.replace("###UNIQUENAME###",child_name).replace("###PACKAGE_NAME###",parent_name.lower()).replace("###SERVICE_OR_QUERY###",info[i]["type"])
      tmp_template = tmp_template.replace(scaffold_methods, 
      METHOD_TEMPLATE.replace("###UNIQUENAME###",child_name).replace("###METHOD_NAME###",info[i]["func_name"]).
      replace("###REQUEST###",info[i]["request"]).
      replace("###RESPONSE###",info[i]["response"]).replace("###WHOLE_SERVICE###",i) + 
      scaffold_methods)
      new_file.write(tmp_template)

REGISTRATION = f"""
package ###PACKAGE_NAME###_thirdparty

import (
	"context"

	"google.golang.org/grpc"
)

func Register###CHAIN###Protobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {"{"}
	{scaffold_register}
{"}"}

{scaffold_registeration}
"""

REGISTER_SINGLE_REGISTRATION = f"""
func Register###CHAIN###Protobufs(s *grpc.Server, cb func(ctx context.Context, method string, reqBody []byte) ([]byte, error)) {"{"}
	{scaffold_register}
{"}"}

"""

ADD_REGISTER = """
###VAR_NAME### := &implemented###UNIQUENAME###{cb: cb}
pkg.Register###SERVER_TYPE###Server(s, ###VAR_NAME###)

"""

for x in childrens:
  register_path = os.path.join(childrens[x]["parent_path"],"register"+ childrens[x]["parent_name"] + "Protobufs" +".go")
  if os.path.exists(register_path):
    with open(register_path,"r") as file_read:
      file_data = file_read.read()
    if (f"Register{chain}Protobufs") in file_data:
      relevant_part = file_data.split(f"Register{chain}Protobufs",1)[-1]
      relevant_part = relevant_part.split(scaffold_registeration,1)[0]
      if childrens[x]["child_name"].lower() in relevant_part:
        continue
      else:
        file_data = file_data.replace(relevant_part,relevant_part.replace(scaffold_register,ADD_REGISTER.replace("###VAR_NAME###",childrens[x]["child_name"].lower())
        .replace("###UNIQUENAME###",childrens[x]["struct_name"]).replace("###SERVER_TYPE###",childrens[x]["server_type"]) + scaffold_register))
    else:
      file_data = file_data.replace(scaffold_registeration, REGISTER_SINGLE_REGISTRATION.replace("###CHAIN###",chain) + scaffold_registeration).replace(scaffold_register,
        ADD_REGISTER.replace("###VAR_NAME###",childrens[x]["child_name"].lower())
        .replace("###UNIQUENAME###",childrens[x]["struct_name"]).replace("###SERVER_TYPE###",childrens[x]["server_type"]) + scaffold_register
        )
    with open(register_path,"w+") as file_write:
      file_write.write(file_data)
  else:
    with open(register_path, "w+") as new_file:
      new_file.write(
        REGISTRATION.replace("###PACKAGE_NAME###", childrens[x]["package_name"]).replace("###CHAIN###",chain).replace(scaffold_register,
        ADD_REGISTER.replace("###VAR_NAME###",childrens[x]["child_name"].lower())
        .replace("###UNIQUENAME###",childrens[x]["struct_name"]).replace("###SERVER_TYPE###",childrens[x]["server_type"]) + scaffold_register
        )
      )

print("finished")