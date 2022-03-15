# Relayer

## Compile protobuf

```bash
# in lava folder
bash ./relayer/compile_proto.sh
```

## Run relayer server

```bash
# in lava folder
go run relayer/cmd/relayer/main.go server
```

## Run relayer test client

```bash
# in lava folder
go run relayer/cmd/relayer/main.go test_client
```
