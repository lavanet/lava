# Mock RPC Server for Smart Router Testing

HTTP server that mimics an Ethereum JSON-RPC endpoint and returns **configurable status codes and delays**. Used to test Section 3 (Direct Relay Behavior & Retry) without depending on real RPCs.

## Run

```bash
# From project root (default port 19999)
go run ./scripts/mock_rpc_server

# Custom port
go run ./scripts/mock_rpc_server -port 19999
```

Ensure the mock is running **before** sending traffic through the Smart Router when `smartrouter_eth.yml` includes the `mock-eth-mainnet` provider (`http://127.0.0.1:19999`).

## Control API (use when testing via Smart Router)

The Smart Router does not forward arbitrary client headers. Set the mock’s response via the control endpoint, then send the RPC through the router.

**When all providers point at the same mock**, the router sends one request per provider in parallel. Use **`sticky=1`** so every request gets the same status until you reset:

```bash
# ALL requests get 500 until you reset (use when router fans out to multiple mock endpoints)
curl -s "http://127.0.0.1:19999/control?status=500&sticky=1"

# Reset to 200 (clears sticky)
curl -s "http://127.0.0.1:19999/control?status=200"
```

Without `sticky`, only the **next** request gets that status, then the mock resets to 200 (so with 5 parallel requests, only 1 gets 500 and the client often sees success).

```bash
# One-shot: next request returns 500, then reset (ok when only 1 provider hits the mock)
curl -s "http://127.0.0.1:19999/control?status=500"

# Next request sleeps 5s (timeout testing)
curl -s "http://127.0.0.1:19999/control?delay=5"
```

### Block consistency (first request X, second request X−200)

Use `block=N` to set the `eth_blockNumber` result (decimal). Smart Router consistency only ever **increases** the seen block; if the second response returns an older block, it is **not** applied.

```bash
# 1) Next response returns block 5000 (0x1388)
curl -s "http://127.0.0.1:19999/control?block=5000"

# 2) Send eth_blockNumber through router → response 5000; consistency becomes 5000
curl -s -X POST -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://127.0.0.1:3360

# 3) Next response returns block 4800 (5000−200)
curl -s "http://127.0.0.1:19999/control?block=4800"

# 4) Send eth_blockNumber again → response 4800; consistency stays 5000 (no update for older block)
curl -s -X POST -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' http://127.0.0.1:3360
```

In logs: first request `requestedBlockForCache=0` then `seenBlock=5000`; second request `requestedBlockForCache=5000` and `seenBlock=5000` (unchanged after 4800 response). No consistency “error” is raised; the older value is simply not written.

## Per-request headers (when calling mock directly)

| Header          | Values                  | Effect                                   |
|-----------------|-------------------------|------------------------------------------|
| `X-Mock-Status` | `200`, `400`, `429`, `500` | HTTP status to return (default `200`)   |
| `X-Mock-Delay`  | seconds, e.g. `5`       | Wait this many seconds before replying  |

## Examples

```bash
# 200 (success) – default
curl -X POST http://127.0.0.1:19999 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 500 – server error (retry, mark unhealthy)
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Status: 500" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 429 – rate limit (retry, do not mark unhealthy)
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Status: 429" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 400 – client error (no retry)
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Status: 400" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 10s delay – timeout testing
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Delay: 10" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

## Config (`smartrouter_eth.yml`)

The first static-provider is the mock:

```yaml
static-providers:
  - name: "mock-eth-mainnet"
    chain-id: "ETH1"
    api-interface: "jsonrpc"
    node-urls:
      - url: "http://127.0.0.1:19999"
  - name: "infura-eth-mainnet"
    # ... real ETH
```

With two providers, when the mock returns 5xx/429/timeout the Smart Router can retry/rotate to Infura. Use `[DirectRPC]` / `[StateMachine] RETRY` logs to confirm behavior.

## Testing via Smart Router (Section 3 checklist)

1. Start mock: `go run ./scripts/mock_rpc_server`
2. Start Smart Router and cache (e.g. `./scripts/pre_setups/init_lava_smartrouter_eth.sh`); use a pre-built `smartrouter_eth.yml` that includes the mock provider.
3. Set mock to fail, then send RPC through the router:

```bash
# 5xx test: mock returns 500 -> router retries to Infura, marks mock unhealthy
curl -s "http://127.0.0.1:19999/control?status=500"
curl -X POST http://127.0.0.1:3360 -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 4xx test: mock returns 400 -> no retry (client error)
curl -s "http://127.0.0.1:19999/control?status=400"
curl -X POST http://127.0.0.1:3360 -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

Watch `tail -f .../SMARTROUTER.log | grep -E '\[DirectRPC\]|\[StateMachine\] RETRY'` to see relay/retry/health behavior.
