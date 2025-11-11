# RPC Consumer (Decentralized Mode)

The RPC Consumer is Lava's **decentralized** RPC gateway that dynamically discovers and pairs with blockchain data providers through on-chain pairing mechanisms. It ensures data reliability through conflict detection, finalization consensus, and multi-provider validation.

## When to Use RPC Consumer

Use **rpcconsumer** when you want:
- ✅ **Decentralized provider discovery** - Automatically find and pair with on-chain staked providers
- ✅ **Data reliability** - Conflict detection and finalization consensus ensure data accuracy
- ✅ **Dynamic provider rotation** - Providers change based on blockchain epochs and staking
- ✅ **Trustless operation** - No need to trust specific provider infrastructure
- ✅ **Full Lava protocol features** - Access all blockchain validation and incentive mechanisms

For **centralized/enterprise deployments** with known provider infrastructure, use [rpcsmartrouter](../rpcsmartrouter) instead.

## Key Features

### Decentralized Architecture
- **On-chain Pairing**: Discovers providers through blockchain pairing lists
- **Epoch Management**: Provider selection updates with blockchain epochs
- **Stake-based Selection**: Providers weighted by their on-chain stake

### Data Reliability
- **Conflict Detection**: Automatically detects and reports provider conflicts
- **Finalization Consensus**: Validates data finalization across multiple providers
- **Response Verification**: Ensures responses match blockchain state

### Advanced Capabilities
- **Provider Optimizer**: QoS-based provider selection with multiple strategies
- **Smart Caching**: Two-layer caching for improved performance
- **Relay Health Monitoring**: Continuous provider health tracking
- **WebSocket Support**: Full support for subscription-based APIs

## Installation

```bash
# Clone the repository
git clone https://github.com/lavanet/lava.git
cd lava

# Install all binaries
make install-all
```

## Configuration

Create a configuration file (e.g., `rpcconsumer.yml`):

```yaml
endpoints:
  - chain-id: ETH1
    api-interface: jsonrpc
    network-address: 127.0.0.1:3333
  - chain-id: OSMOSIS
    api-interface: rest
    network-address: 127.0.0.1:3334
  - chain-id: OSMOSIS
    api-interface: tendermintrpc
    network-address: 127.0.0.1:3335
```

### Configuration Fields

- `chain-id`: Blockchain identifier (e.g., ETH1, OSMOSIS, LAV1)
- `api-interface`: API type (jsonrpc, rest, tendermintrpc, grpc)
- `network-address`: IP:PORT where the consumer will listen for requests

## Usage

### Basic Usage

```bash
lavap rpcconsumer rpcconsumer.yml \
  --from wallet_name \
  --chain-id lava-testnet-2 \
  --geolocation 1
```

### Required Flags

- `--from`: Your wallet name (used for signing relay requests and conflict detection)
- `--chain-id`: Lava blockchain chain ID (e.g., lava-testnet-2, lava-mainnet-1)
- `--geolocation`: Your geographic location code

### Common Options

```bash
lavap rpcconsumer rpcconsumer.yml \
  --from wallet_name \
  --chain-id lava-testnet-2 \
  --geolocation 1 \
  --cache-be "127.0.0.1:7778" \
  --strategy latency \
  --metrics-listen-address ":7779" \
  --log_level debug
```

### Strategy Options

Choose provider selection strategy with `--strategy`:
- `balanced` (default) - Balance between all factors
- `latency` - Prioritize fastest providers
- `sync-freshness` - Prioritize most up-to-date providers
- `cost` - Optimize for lower cost
- `privacy` - Maximize privacy
- `accuracy` - Prioritize data accuracy
- `distributed` - Distribute across many providers

### Advanced Flags

```bash
# Enable caching
--cache-be "127.0.0.1:7778"

# Prometheus metrics
--metrics-listen-address ":7779"

# Kafka analytics
--relay-kafka-addr "localhost:9092"

# Optimizer QoS reports
--optimizer-qos-server-address "http://qos-server:8080"

# Debug options
--debug-relays
--log_level trace

# Performance tuning
--max-concurrent-providers 5
```

## Example Requests

After starting the consumer, make RPC requests to the configured endpoints:

```bash
# REST API request
curl http://127.0.0.1:3334/cosmos/base/tendermint/v1beta1/blocks/latest

# JSON-RPC request
curl -X POST http://127.0.0.1:3333 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }'

# WebSocket subscription
wscat -c ws://127.0.0.1:3333
> {"jsonrpc":"2.0","id":1,"method":"eth_subscribe","params":["newHeads"]}
```

## How It Works

1. **Blockchain Pairing**: Consumer queries Lava blockchain for provider pairings
2. **Provider Selection**: Selects providers based on stake, QoS, and strategy
3. **Request Routing**: Routes user requests to selected providers
4. **Response Validation**: Validates responses against finalization consensus
5. **Conflict Detection**: Reports any conflicts to the blockchain
6. **QoS Updates**: Continuously updates provider QoS scores

## Architecture

```
User Request → RPC Consumer → Blockchain Pairing Query
                     ↓
              Provider Selection (stake-weighted)
                     ↓
              Relay to Providers (parallel)
                     ↓
              Response Validation & Consensus
                     ↓
              Conflict Detection (if needed)
                     ↓
              Return Best Response
```

## vs. RPC Smart Router

| Feature | RPC Consumer (Decentralized) | RPC Smart Router (Centralized) |
|---------|------------------------------|--------------------------------|
| Provider Discovery | On-chain blockchain pairing | Static configuration file |
| Trust Model | Trustless (blockchain verified) | Trust known providers |
| Provider Selection | Stake-weighted + QoS | QoS only |
| Data Validation | Conflict detection + consensus | Basic validation |
| Setup Complexity | Medium (requires wallet) | Simple (just config file) |
| Use Case | Public networks, trustless | Enterprises, known infrastructure |
| Blockchain Dependency | Yes (requires Lava chain) | Optional (only for specs) |

## Monitoring & Metrics

RPC Consumer exposes Prometheus metrics at the configured `--metrics-listen-address`:

```bash
# View metrics
curl http://localhost:7779/metrics

# Key metrics:
# - lava_consumer_relay_requests_total
# - lava_consumer_relay_errors_total
# - lava_consumer_provider_qos_score
# - lava_consumer_cache_hit_rate
```

## Troubleshooting

### "No pairings available"
- Ensure your wallet has an active subscription
- Check that providers are staked for your requested chain
- Verify geolocation matches available providers

### "Failed to get provider address"
- Check blockchain connectivity
- Verify chain-id is correct
- Ensure subscription is active

### Performance Issues
- Enable caching with `--cache-be`
- Adjust `--max-concurrent-providers`
- Use `--strategy latency` for speed
- Check provider QoS metrics

## Configuration Examples

See the `config/consumer_examples/` directory for complete configuration examples:
- `full_consumer_example.yml` - All features enabled
- `ethereum_example.yml` - Ethereum-specific setup
- `osmosis_example.yml` - Cosmos chain example

## More Information

- [Lava Documentation](https://docs.lavanet.xyz)
- [Consumer Configuration Guide](../../config/consumer_examples/)
- [RPC Smart Router](../rpcsmartrouter) - Centralized alternative
- [Protocol Overview](../README.md)
