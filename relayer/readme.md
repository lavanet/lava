# Relayer

## Compile protobuf

```bash
# in lava folder
bash ./relayer/compile_proto.sh
```

## Run relayer server

```bash
# in lava folder
go run relayer/cmd/relayer/main.go server 127.0.0.1 2222 wss://mainnet.infura.io/ws/v3/<your_token> 0 --from bob
```

## Run relayer test client

```bash
# in lava folder
go run relayer/cmd/relayer/main.go test_client 127.0.0.1 2222 --from alice
```
