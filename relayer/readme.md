# Relayer

## Compile protobuf

```bash
# in lava folder
bash ./relayer/compile_proto.sh
```

## Run relayer server

```bash
# in lava folder
lavad server 127.0.0.1 2222 wss://mainnet.infura.io/ws/v3/<your_token> 0 --from bob
```

## Run relayer test client

```bash
# in lava folder
lavad test_client 0 --from alice
```

## Run portal server

```bash
# in lava folder
lavad portal_server 127.0.0.1 3333 0 --from user2
geth attach ws://127.0.0.1:3333/ws
```
### debug
for a more verbose logging use the flag: --log_level debug
## Debug the relayer mutexes

This flag turns on warnings for mutexes thay are locked for a long time
```bash
# in lava folder
DEBUG_MUTEX="true" make # make with this flag on
# Run any of the above with the compiled lavad
build/lavad server 127.0.0.1 2222 wss://mainnet.infura.io/ws/v3/<your_token> 0 --from bob
```

## Hide Portal Errors for consumer requests

This flag will show only a unique identifier id for each error. 

If the flag is true:
``` 
curl -X GET "http://127.0.0.1:3340/1/nbobo"
{"error": "unsupported api","more_information" Error guid: GUID2756376310285318670}% 
```

If the flag is off
```
curl -X GET "http://127.0.0.1:3340/1/nbobo"
{"error": "unsupported api","more_information" Error guid: GUID55979968042711362, Error: REST Api not supported /nbobo }%
```

To run the flag use the make file with the following command

off:
```
MASK_CONSUMER_LOGS="false"; make build
```
on:
```
MASK_CONSUMER_LOGS="false"; make build
```