# RPC Smart Router (Centralized Mode)

The RPC Smart Router is Lava's **centralized** RPC gateway designed for enterprises with known provider infrastructure. It routes requests to pre-configured static and backup providers without requiring blockchain pairing, making it ideal for controlled environments.

**Trusted by:** Fireblocks, Movement, Arbitrum, NEAR, Filecoin, Cosmos, and many more enterprise teams.

## When to Use RPC Smart Router

Use **rpcsmartrouter** when you want:
- ✅ **Known provider infrastructure** - You control or trust specific RPC endpoints
- ✅ **Simple setup** - No blockchain wallet or pairing required
- ✅ **Predictable routing** - Providers don't change based on blockchain epochs
- ✅ **Enterprise control** - Full control over which providers are used
- ✅ **Backup failover** - Configure primary and backup provider tiers
- ✅ **Performance optimization** - QoS-based routing without blockchain overhead

For **decentralized/trustless** operation with on-chain provider discovery, use [rpcconsumer](../rpcconsumer) instead.

## Key Features

### Static Provider Configuration
- **Pre-configured Providers**: Define your trusted RPC endpoints in config
- **Backup Providers**: Automatic failover to backup tier when primaries fail
- **No Blockchain Dependency**: Runs without on-chain pairing (specs can be static)
- **Multi-provider Support**: Mix Alchemy, Infura, self-hosted, and other providers

### Intelligent Routing
- **QoS-based Selection**: Routes to best-performing providers
- **Automatic Failover**: Seamlessly switches to backups on provider failure
- **Health Monitoring**: Continuous provider health checks
- **Strategy Options**: Multiple routing strategies (latency, balanced, etc.)

### Enterprise Features
- **Smart Caching**: Two-layer caching reduces provider load
- **Transaction Broadcasting**: Sends transactions to all providers for faster propagation
- **WebSocket Support**: Full support for subscription-based APIs
- **Metrics & Monitoring**: Prometheus metrics and health endpoints
- **Analytics Integration**: Kafka support for detailed analytics

## Installation

```bash
# Clone the repository
git clone https://github.com/lavanet/lava.git
cd lava

# Install all binaries
make install-all
```

## Configuration

Create a configuration file (e.g., `rpcsmartrouter.yml`):

```yaml
endpoints:
  - chain-id: ETH1
    api-interface: jsonrpc
    network-address: 127.0.0.1:3333
  - chain-id: OSMOSIS
    api-interface: rest
    network-address: 127.0.0.1:3334

static-providers:
  # Primary providers (used first)
  - name: alchemy-eth-mainnet
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
  
  - name: infura-eth-mainnet
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://mainnet.infura.io/v3/YOUR_KEY
  
  - name: self-hosted-osmosis
    chain-id: OSMOSIS
    api-interface: rest
    node-urls:
      - url: http://osmosis-node-1.internal:1317

backup-providers:
  # Backup providers (used when primaries fail)
  - name: backup-alchemy
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://eth-mainnet.g.alchemy.com/v2/YOUR_BACKUP_KEY
  
  - name: backup-osmosis
    chain-id: OSMOSIS
    api-interface: rest
    node-urls:
      - url: http://osmosis-node-2.internal:1317
```

### Configuration Fields

#### Endpoints
- `chain-id`: Blockchain identifier
- `api-interface`: API type (jsonrpc, rest, tendermintrpc, grpc)
- `network-address`: IP:PORT where smart router listens

#### Static Providers (Primary)
- `name`: Unique identifier for this provider
- `chain-id`: Which blockchain this provider serves
- `api-interface`: API type
- `node-urls`: List of RPC endpoint URLs
  - `url`: The actual RPC endpoint (http://, https://, ws://)
  - `addons`: Optional extensions (e.g., archive, debug, trace)

#### Backup Providers (Failover)
- Same format as static providers
- Used automatically when all static providers fail
- Optional but **highly recommended** for production

## Usage

### Basic Usage

```bash
lavap rpcsmartrouter rpcsmartrouter.yml \
  --geolocation 1
```

### Required Flags

- `--geolocation`: Your geographic location code (for metrics/routing)

### Common Options

```bash
lavap rpcsmartrouter rpcsmartrouter.yml \
  --geolocation 1 \
  --cache-be "127.0.0.1:7778" \
  --strategy latency \
  --metrics-listen-address ":7779" \
  --log_level debug
```

### Strategy Options

Choose provider selection strategy with `--strategy`:
- `balanced` (default) - Balance between latency and reliability
- `latency` - Prioritize fastest providers
- `sync-freshness` - Prioritize most up-to-date providers
- `cost` - Optimize for lower cost (if configured)
- `privacy` - Maximize privacy distribution
- `distributed` - Distribute load across many providers

### Advanced Flags

```bash
# Enable caching
--cache-be "127.0.0.1:7778"

# Prometheus metrics
--metrics-listen-address ":7779"

# Kafka analytics
--relay-kafka-addr "localhost:9092"
--relay-kafka-topic "lava-relay-metrics"

# Performance tuning
--max-concurrent-providers 5

# Debug options
--debug-relays
--log_level trace

# Load static spec (no blockchain needed)
--use-static-spec-file "./specs/eth.json"
```

## Example Requests

After starting the smart router, make RPC requests to configured endpoints:

```bash
# JSON-RPC request (routed to best ETH provider)
curl -X POST http://127.0.0.1:3333 \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
  }'

# REST API request (routed to best Osmosis provider)
curl http://127.0.0.1:3334/cosmos/base/tendermint/v1beta1/blocks/latest

# With failover - if primary fails, automatically uses backup
curl http://127.0.0.1:3333 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x123...",  "latest"],"id":1}'
```

## How It Works

1. **Configuration Loading**: Reads static and backup providers from config
2. **Provider Initialization**: Connects to all configured providers
3. **Health Monitoring**: Continuously monitors provider health
4. **Request Routing**: Routes requests to best available provider
5. **QoS Tracking**: Tracks latency, errors, and success rates
6. **Automatic Failover**: Switches to backup providers on failure
7. **Load Balancing**: Distributes load based on QoS scores

## Architecture

```
User Request → Smart Router → Provider Selection (QoS-based)
                     ↓
              Try Static Providers
                     ↓
              [If all fail] → Try Backup Providers
                     ↓
              Cache Response (optional)
                     ↓
              Return to User
```

## Multi-Provider Example

Configure multiple providers for high availability:

```yaml
static-providers:
  # Mix of different provider types
  - name: alchemy-primary
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://eth-mainnet.g.alchemy.com/v2/PRIMARY_KEY

  - name: infura-primary
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://mainnet.infura.io/v3/PRIMARY_KEY
  
  - name: quicknode-primary
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://your-endpoint.quiknode.pro/YOUR_KEY

  - name: self-hosted-archive
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: http://archive-node.internal:8545
        addons: ["archive", "debug", "trace"]

backup-providers:
  - name: alchemy-backup
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://eth-mainnet.g.alchemy.com/v2/BACKUP_KEY
```

## vs. RPC Consumer

| Feature | RPC Smart Router (Centralized) | RPC Consumer (Decentralized) |
|---------|--------------------------------|------------------------------|
| Provider Discovery | Static configuration | On-chain blockchain pairing |
| Setup Complexity | Simple (just config) | Medium (requires wallet) |
| Trust Model | Trust known providers | Trustless (blockchain verified) |
| Provider Selection | QoS only | Stake-weighted + QoS |
| Data Validation | Basic validation | Conflict detection + consensus |
| Blockchain Dependency | Optional (only for specs) | Required |
| Backup Providers | ✅ Built-in failover | ❌ Not applicable |
| Best For | Enterprises, known infra | Public networks, trustless |

## Backup Provider Failover

The smart router automatically fails over to backup providers:

1. **Primary Attempt**: Tries static providers first (best QoS selected)
2. **Failure Detection**: Detects errors, timeouts, or unavailability
3. **Automatic Failover**: Switches to backup providers transparently
4. **Recovery**: Monitors primary providers and switches back when healthy
5. **Logging**: All failover events are logged for monitoring

Example log output:
```
INF Configured backup providers for endpoint chainID=ETH1 apiInterface=jsonrpc backupCount=2
WRN Primary provider failed, trying backup provider=alchemy-eth-mainnet error="connection timeout"
INF Successfully failed over to backup provider=backup-alchemy-eth
```

## Monitoring & Metrics

Smart Router exposes Prometheus metrics:

```bash
# View metrics
curl http://localhost:7779/metrics

# Key metrics:
# - lava_smart_router_requests_total
# - lava_smart_router_provider_qos_score
# - lava_smart_router_failover_events_total
# - lava_smart_router_cache_hit_rate
# - lava_smart_router_provider_latency_seconds
```

## Health Checks

Built-in health monitoring:

```bash
# Enable health checks (enabled by default)
--relays-health-enable=true
--relay-health-interval=5m

# Health check logs
DBG Health check sent to provider provider=alchemy-primary success=true latency=45ms
```

## Troubleshooting

### "No static providers configured"
- Check `static-providers` section in config
- Ensure chain-id matches endpoint chain-id
- Verify api-interface matches endpoint api-interface

### All Providers Failing
- Check provider URLs are accessible
- Verify API keys are valid
- Check network connectivity
- Review backup provider configuration

### Performance Issues
- Enable caching: `--cache-be "127.0.0.1:7778"`
- Increase concurrent providers: `--max-concurrent-providers 5`
- Use latency strategy: `--strategy latency`
- Check provider QoS metrics

### Backup Providers Not Working
- Ensure backup providers are configured
- Verify backup URLs are different from static
- Check logs for failover events

## Configuration Examples

See `config/consumer_examples/` directory:
- `lava_consumer_static_with_backup.yml` - Full example with backups
- `lava_consumer_static_peers.yml` - Simple static provider setup

## Production Deployment

### Recommended Setup

```yaml
# Production configuration with redundancy
endpoints:
  - chain-id: ETH1
    api-interface: jsonrpc
    network-address: 0.0.0.0:3333  # Bind to all interfaces

static-providers:
  # Use multiple providers for redundancy
  - name: provider-1
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://provider1.example.com/v1/YOUR_KEY

  - name: provider-2
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://provider2.example.com/v1/YOUR_KEY

backup-providers:
  # Different provider for true failover
  - name: backup-provider
    chain-id: ETH1
    api-interface: jsonrpc
    node-urls:
      - url: https://backup.example.com/v1/YOUR_KEY
```

### Production Flags

```bash
lavap rpcsmartrouter config.yml \
  --geolocation 1 \
  --cache-be "redis:7778" \
  --metrics-listen-address "0.0.0.0:7779" \
  --relay-kafka-addr "kafka:9092" \
  --log_level info \
  --max-concurrent-providers 5 \
  --strategy balanced
```

## More Information

- [Lava Documentation](https://docs.lavanet.xyz)
- [Configuration Examples](../../config/consumer_examples/)
- [RPC Consumer](../rpcconsumer) - Decentralized alternative
- [Protocol Overview](../README.md)
