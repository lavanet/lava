Automation Scripts Readme

# gRPC Spec Builder: 

Used for verifying the REST part of a spec, and building the grpc part from it.
It uses the grpc descriptors of the endpoint you provide in order to cross validate the missing parts, and the grpc spec itself.

### Usage: 

* Set the following parameters before use, for example:
```
grpc_server = "juno-node-1.lavapro.xyz:9090"
spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_juno.json" 
```

Launch the script

```
python3 scripts/automation_scripts/grpc_spec_builder.py
```

Read the results carefully to understand if any Rest API's were missing / more than necessary. take a look at the new grpc spec file that was created 

# gRPC Scaffolder: 

Used for Scaffolding grpc interface. 

### Usage:

* Set the following parameters before use, for example:
```
grpc_server = "juno-node-1.lavapro.xyz:9090"
spec_current_file_path = "/home/user/go/src/lava/cookbook/spec_add_juno.json"
chain = "Juno"
result_dir = "/home/user/go/src/lava/scripts/automation_scripts/automation_results/grpcClientProtobufs"
```

Launch the script, if grpc files are already located in there it will just append the missing API's. 

* There might be bugs in the script as it wasnt tested enough, so make sure you have a backed up version from git. its alot easier to compare the diff afterwards 

```
python3 scripts/automation_scripts/grpc_scaffolder.py
```