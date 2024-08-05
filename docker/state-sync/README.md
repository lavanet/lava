# Running node using state-sync with docker-compose

From the root path run:

```sh
docker compose -f docker/state-sync/docker-compose.yml up -d
```

To test the setup run:

```sh
curl -X POST -H "Content-Type: application/json" localhost:26657 --data '{"jsonrpc": "2.0", "id": 1, "method": "status", "params": []}'
```

and expect to see the lastest block.

You can run change the version of `lavad` using the `LAVAD_VERSION` var:

```sh
LAVAD_VERSION=v2.0.1 docker compose -f docker/state-sync/docker-compose.yml up -d
```

## Full configuration options

|Name            |Description
|----------------|-------------------------------
|LAVAD_VERSION   | The Lavad version to use
|CHAIN_ID        | The chain id
|KEYRING_BACKEND | The keyring backend
|MONIKER         | The moniker for the `init` command
|STATE_SYNC_RPC_1| The RPC node to sync on
|GENESIS_ADDRESS | The `genesis.json` URL
|ADDRBOOK_ADDRESS| The `addrbook.json` URL
|NUM_BLOCKS      | The number of blocks to sync on from behind the latest block

To clean the lava node setup including volumes run:

```sh
docker compose -f docker/state-sync/docker-compose.yml down -v
```
