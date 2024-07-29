# Lava docker support

Lava Network offers Docker support to simplify deployment and management of its nodes and services. Key features include:

* **Containerized Environment**: Run Lava components in isolated containers for improved consistency and portability.
* **Easy Setup**: Quickly deploy Lava nodes using pre-configured Docker images.
* **Scalability**: Easily scale your Lava infrastructure by spinning up additional containers as needed.
* **Resource Efficiency**: Optimize resource usage by running multiple Lava services on a single host.
* **Cross-platform Compatibility**: Deploy Lava nodes consistently across different operating systems and environments.

## Different Lava configuration setups

The compose files are ordered in sub-folders and can be simply run with:

```shell
docker compose -f <compose-file> up -d
```

## Running Lava containers with docker-compose

The best way to deploy the Lava echo-system is via docker-compose.

*Requirments*:

* Docker Compose v2

To start using the compose files see the examples under the `docker/` directory:

* `/state-sync` - running a node by [state-syncing](https://docs.tendermint.com/v0.34/tendermint-core/state-sync.html) with another.
* `/from-snapshot` - running a node by downloading an exising snapshot and syncing.
* `/new-node` - running a fresh node from scratch.
* `/load-balancing` - running multiple providers load-balanced by Nginx proxy

## Building Lava docker images

In order to buid the Lava docker image follow these steps:

1. Download the lava sources:

   ```shell
   git clone https://github.com/lavanet/lava.git
   ```

2. Build the appropriate Lava docker image locally

   ```shell
   docker buildx build -f cmd/lavad/Dockerfile .
   docker buildx build -f cmd/lavad/Dockerfile.Cosmovisor .
   docker buildx build -f cmd/lavap/Dockerfile .
   ```
   