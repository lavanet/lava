# Lava docker support

Lava Network offers Docker support to simplify deployment and management of its nodes and services. Key features include:
* **Containerized Environment**: Run Lava components in isolated containers for improved consistency and portability.
* **Easy Setup**: Quickly deploy Lava nodes using pre-configured Docker images.
* **Scalability**: Easily scale your Lava infrastructure by spinning up additional containers as needed.
* **Resource Efficiency**: Optimize resource usage by running multiple Lava services on a single host.
* **Cross-platform Compatibility**: Deploy Lava nodes consistently across different operating systems and environments.

## Different Lava configuration setups

The compose files are ordered in sub-folders and can be simply run with:
```
docker compose -f <compose-file> up -d
```

Details about the different configurations:

|Name            |Description
|----------------|-------------------------------
|`state-sync`      | Setup Lava node via state sync from other node


## Building Lava docker images

In order to buid the Lava docker image follow these steps:

1. Download the lava sources:

   ```
   git clone https://github.com/lavanet/lava.git
   ```

2. Build the appropriate Lava docker image locally

   ```
   docker buildx build -f cmd/lavad/Dockerfile .
   docker buildx build -f cmd/lavad/Dockerfile.Cosmovisor .
   docker buildx build -f cmd/lavap/Dockerfile .
   docker buildx build -f cmd/lavavisor/Dockerfile .
   ```

  The result would be a docker image names `lava` tagged with the version.
  For example the output of `docker images` after the above:

  ```
  lava             0.4.3                 bc3a85c7623f   2 hours ago      256MB
  lava             latest                bc3a85c7623f   2 hours ago      256MB
  lava             0.4.3-a5e1202-dirty   5ff644084c3d   2 hours ago      257MB
  ```

## Running lava containers with docker

**TODO**

## Running lava containers with docker-compose

**Run Lava Node**

1. Review the settings in `docker/env` (sections "common setup" and "common
runtime"). The default settings are usually suitable for all deployments.

2. Use the following the commands to create/start/stop/destroy the node:

  ```
  # operate in docker/ directory:
  cd docker/

  # to start the node:
  docker-compose --profile node --env-file env -f docker-compose.yml up

  # to stop/restart the node:
  docker-compose --profile node --env-file env -f docker-compose.yml stop
  docker-compose --profile node --env-file env -f docker-compose.yml start

  # to destroy the node:
  docker-compose --profile node --env-file env -f docker-compose.yml down
  ```

## Running node using state-sync with docker-compose

From the root path run:

```sh
docker compose -f docker/docker-compose.state-sync.yml up -d
```

To test the setup run:

```sh
curl -X POST -H "Content-Type: application/json" localhost:26657 --data '{"jsonrpc": "2.0", "id": 1, "method": "status", "params": []}'
```

and expect to see the lastest block.

You can run change the version of `lavad` using the `LAVAD_VERSION` var:

```sh
LAVAD_VERSION=v2.0.1 docker compose -f docker/docker-compose.state-sync.yml up -d
```

For `lavad` image with [cosmovisor](https://github.com/cosmos/cosmos-sdk/tree/main/tools/cosmovisor) support run:

```sh
LAVAD_VERSION=v2.0.1 docker compose -f docker/docker-compose.state-sync.cosmovisor.yml up -d
```

### Full configuration options

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
docker compose -f docker/docker-compose.state-sync.yml down -v
```
