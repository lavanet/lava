# Lava EVM Indexer

A high-performance blockchain indexing service for EVM-compatible chains that integrates with Lava's smart router.

## Features

- **Block Indexing**: Continuously indexes blocks, transactions, and receipts
- **Smart Contract Events**: Monitor and index specific contract events
- **REST API**: Query indexed data via a clean REST API
- **Multiple Storage Backends**: SQLite (with PostgreSQL support coming soon)
- **Real-time Monitoring**: Built-in metrics and health checks
- **Reorg Handling**: Detects and handles blockchain reorganizations
- **High Performance**: Concurrent workers and batch processing
- **Smart Router Integration**: Works seamlessly with Lava's smart router

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Lava EVM Indexer                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │    Block     │  │   Contract   │  │   API Server    │  │
│  │   Indexer    │  │   Watcher    │  │   (REST API)    │  │
│  └──────┬───────┘  └──────┬───────┘  └────────┬────────┘  │
│         │                 │                    │           │
│         └─────────────────┴────────────────────┘           │
│                           │                                │
│                  ┌────────▼────────┐                       │
│                  │  Storage Layer  │                       │
│                  │  (SQLite/PG)    │                       │
│                  └─────────────────┘                       │
│                           ▲                                │
└───────────────────────────┼────────────────────────────────┘
                            │
                  ┌─────────▼─────────┐
                  │   RPC Endpoint    │
                  │ (Smart Router or  │
                  │   Direct Node)    │
                  └───────────────────┘
```

## Quick Start

### 1. Installation

```bash
# Clone the repository
git clone https://github.com/lavanet/lava.git
cd lava

# Install all binaries
make install-all
```

### 2. Generate Example Configuration

```bash
lavap indexer --example-config > indexer.yml
```

### 3. Edit Configuration

Edit `indexer.yml` to customize:

```yaml
# RPC Configuration
rpc_endpoint: "http://localhost:3333"  # Smart router or direct RPC endpoint
chain_id: "ETH1"

# Database Configuration
database_type: "sqlite"
database_url: "indexer.db"

# Indexing Configuration
start_block: 0                # Block to start indexing from (0 = latest)
batch_size: 10                # Blocks per batch
concurrent_workers: 4         # Parallel workers
polling_interval: 12s         # How often to check for new blocks
confirmation_blocks: 0        # Wait for N confirmations
reorg_depth: 10              # Blocks to check for reorgs
index_transactions: true      # Index full transaction data
index_logs: true             # Index event logs
index_full_blocks: true      # Get full block with transactions

# API Server Configuration
enable_api: true
api_listen_addr: ":8080"
enable_metrics: true
metrics_addr: ":9090"

# Contract Watch Configuration (optional)
watched_contracts:
  - name: "USDC"
    address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
    start_block: 6082465
    event_filter:
      - "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"  # Transfer

# Performance Configuration
cache_enabled: true
cache_size: 100
cache_ttl: 5m
max_retries: 3
retry_delay: 1s
```

### 4. Run the Indexer

```bash
lavap indexer indexer.yml
```

## Usage with Smart Router

The indexer works seamlessly with Lava's smart router. First, start the smart router:

```bash
lavap rpcsmartrouter smartrouter.yml --geolocation 1
```

Then configure the indexer to use the smart router endpoint:

```yaml
rpc_endpoint: "http://localhost:3333"  # Smart router endpoint
chain_id: "ETH1"
```

This setup provides:
- **Automatic Failover**: Smart router handles provider failures
- **Load Balancing**: Distributes requests across multiple providers
- **QoS Optimization**: Routes to best-performing providers
- **Reduced Costs**: Leverages Lava's provider network

## REST API

### Blocks

```bash
# Get latest block
curl http://localhost:8080/api/v1/blocks/latest

# Get specific block
curl http://localhost:8080/api/v1/blocks/12345

# Query blocks
curl "http://localhost:8080/api/v1/blocks?from=1000&to=2000&limit=10"

# Filter by miner
curl "http://localhost:8080/api/v1/blocks?miner=0x..."
```

### Transactions

```bash
# Get specific transaction
curl http://localhost:8080/api/v1/transactions/0x123...

# Query transactions
curl "http://localhost:8080/api/v1/transactions?from_block=1000&to_block=2000"

# Filter by address
curl "http://localhost:8080/api/v1/transactions?from=0x...&limit=100"

# Find contract creations
curl "http://localhost:8080/api/v1/transactions?contract=0x..."
```

### Logs (Events)

```bash
# Get logs for a contract
curl "http://localhost:8080/api/v1/logs?contract=0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"

# Filter by block range
curl "http://localhost:8080/api/v1/logs?from_block=1000&to_block=2000"

# Filter by topics (event signatures)
curl "http://localhost:8080/api/v1/logs?topic0=0xddf252ad..."
```

### Contract Watches

```bash
# List watched contracts
curl http://localhost:8080/api/v1/watches
```

### Status & Metrics

```bash
# Health check
curl http://localhost:8080/health

# Indexer status
curl http://localhost:8080/api/v1/status

# Metrics
curl http://localhost:8080/metrics
```

## Configuration Reference

### RPC Configuration

| Field | Type | Description |
|-------|------|-------------|
| `rpc_endpoint` | string | RPC endpoint URL (smart router or direct node) |
| `chain_id` | string | Chain identifier (e.g., "ETH1", "POLYGON1") |

### Database Configuration

| Field | Type | Description |
|-------|------|-------------|
| `database_type` | string | Database type ("sqlite", "postgres") |
| `database_url` | string | Connection string or file path |

### Indexing Configuration

| Field | Type | Description |
|-------|------|-------------|
| `start_block` | int64 | Starting block (0 = latest) |
| `batch_size` | int | Blocks per batch |
| `concurrent_workers` | int | Number of parallel workers |
| `polling_interval` | duration | Polling frequency |
| `confirmation_blocks` | int | Wait for N confirmations |
| `reorg_depth` | int | Blocks to check for reorgs |
| `index_transactions` | bool | Index transaction data |
| `index_logs` | bool | Index event logs |
| `index_full_blocks` | bool | Fetch full block data |

### API Server Configuration

| Field | Type | Description |
|-------|------|-------------|
| `enable_api` | bool | Enable REST API server |
| `api_listen_addr` | string | API server address |
| `enable_metrics` | bool | Enable metrics endpoint |
| `metrics_addr` | string | Metrics server address |

### Contract Watch Configuration

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Contract name |
| `address` | string | Contract address |
| `start_block` | int64 | Start indexing from block |
| `event_filter` | []string | Event signatures to index |

## Advanced Usage

### Indexing Specific Contracts

To index only specific contracts:

```yaml
watched_contracts:
  - name: "Uniswap V2 Router"
    address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
    start_block: 10207858
    event_filter:
      - "0xd78ad95f..." # Swap event
      
  - name: "WETH"
    address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
    start_block: 4719568
    event_filter:
      - "0xddf252ad..." # Transfer
      - "0xe1fffcc4..." # Deposit
      - "0x7fcf532c..." # Withdrawal
```

### Historical Indexing

To index from a specific block:

```yaml
start_block: 15000000  # Start from block 15M
batch_size: 100        # Larger batches for historical data
concurrent_workers: 8  # More workers for faster sync
```

### Monitoring Setup

Enable Prometheus metrics:

```yaml
enable_metrics: true
metrics_addr: ":9090"
```

Then configure Prometheus to scrape:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'lava-indexer'
    static_configs:
      - targets: ['localhost:9090']
```

## Performance Tuning

### For Fast Sync (Historical Data)

```yaml
batch_size: 100
concurrent_workers: 8
confirmation_blocks: 0
cache_enabled: true
cache_size: 500
```

### For Real-time Indexing

```yaml
batch_size: 1
concurrent_workers: 2
confirmation_blocks: 12
polling_interval: 6s
```

### For Event-Heavy Contracts

```yaml
index_transactions: false  # Skip if not needed
index_logs: true
watched_contracts: [...]  # Specify contracts
```

## Troubleshooting

### Indexer Falls Behind

1. Increase `concurrent_workers`
2. Increase `batch_size`
3. Reduce `confirmation_blocks`
4. Enable caching

### High Memory Usage

1. Reduce `concurrent_workers`
2. Reduce `batch_size`
3. Reduce `cache_size`

### Missing Logs

1. Ensure `index_logs: true`
2. Check `watched_contracts` configuration
3. Verify `start_block` is correct

### RPC Errors

1. Check RPC endpoint connectivity
2. Increase `max_retries`
3. Increase `retry_delay`
4. Use smart router for better reliability

## Example: Indexing DeFi Protocols

```yaml
rpc_endpoint: "http://localhost:3333"
chain_id: "ETH1"
database_type: "sqlite"
database_url: "defi_indexer.db"

start_block: 0
index_transactions: true
index_logs: true

watched_contracts:
  # Uniswap V2
  - name: "Uniswap V2 Factory"
    address: "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"
    start_block: 10000835
    
  - name: "Uniswap V2 Router"
    address: "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
    start_block: 10207858
    
  # Aave V2
  - name: "Aave Lending Pool"
    address: "0x7d2768dE32b0b80b7a3454c06BdAc94A69DDc7A9"
    start_block: 11362579
    
  # Compound
  - name: "cDAI"
    address: "0x5d3a536E4D6DbD6114cc1Ead35777bAB948E3643"
    start_block: 8983575
```

## Development

### Running Tests

```bash
cd protocol/indexer
go test -v ./...
```

### Building

```bash
make install-all
```

## Contributing

Contributions are welcome! Please see [CODING_GUIDELINES.md](../../CODING_GUIDELINES.md).

## License

See [LICENSE.md](../../LICENSE.md).
