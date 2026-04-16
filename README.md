# Lava Smart Router

RPC routing solution for the Lava protocol. Routes JSON-RPC, REST, gRPC, and Tendermint RPC requests to statically configured provider endpoints with QoS-based selection, caching, and automatic failover.

## Quick Start

### Prerequisites

- [Go 1.26+](https://go.dev/dl/)

### Build

```bash
make install-all
```

This installs two binaries:
- `smartrouter` — the main smart router binary
- `lavap` — the Lava protocol binary (includes smart router as a subcommand)

### Run

```bash
# Using the standalone binary
smartrouter smartrouter_lava.yml --geolocation 1 --use-static-spec specs/

# Using lavap
lavap rpcsmartrouter smartrouter_lava.yml --geolocation 1 --use-static-spec specs/
```

### Configuration

Provider endpoints are configured in a YAML file. See `smartrouter_lava.yml` for an example with Lava mainnet via PublicNode.

Setup scripts are available in `scripts/pre_setups/`:

```bash
# Lava (REST + gRPC + Tendermint RPC)
./scripts/pre_setups/init_lava_smartrouter_lava.sh

# Ethereum (JSON-RPC)
./scripts/pre_setups/init_lava_smartrouter_eth.sh
```

## Supported Chains

Specs are in the `specs/` directory:

| Chain | Index | Interfaces |
|-------|-------|------------|
| Lava | LAVA | REST, gRPC, TendermintRPC |
| Ethereum | ETH1 | JSON-RPC |

## Development

### Build targets

```bash
make build          # Build smartrouter binary to build/
make build-all      # Build all binaries to build/
make test           # Run all tests
make test-short     # Run smart router tests only
make lint           # Run go vet
make tidy           # Run go mod tidy
make clean          # Remove build artifacts
```

### Project structure

```
cmd/smartrouter/    — Standalone smart router binary
cmd/lavap/          — Lava protocol binary (includes smart router)
protocol/           — Core protocol implementation
  chainlib/         — Chain-specific parsers and proxies
  rpcsmartrouter/   — Smart router server and relay logic
  lavasession/      — Session and connection management
  provideroptimizer — QoS-based provider selection
  relaycore/        — Relay processing pipeline
  metrics/          — Prometheus metrics
types/              — Shared type definitions
specs/              — Chain specification JSON files
```

## Community

- [Discord](https://discord.com/invite/Tbk5NxTCdA)
- [X (formerly Twitter)](https://x.com/lavanetxyz)
