# Mock RPC Server

HTTP server that mimics an Ethereum JSON-RPC endpoint and returns **configurable status codes and delays**. Useful for testing relay behavior and retry logic without depending on real RPCs.

## Run

```bash
# From project root (default port 19999)
go run ./scripts/mock_rpc_server

# Custom port
go run ./scripts/mock_rpc_server -port 19999
```

## Control API

Set the mock's response behavior via the control endpoint:

```bash
# ALL requests get 500 until you reset (sticky mode)
curl -s "http://127.0.0.1:19999/control?status=500&sticky=1"

# Reset to 200 (clears sticky)
curl -s "http://127.0.0.1:19999/control?status=200"
```

Without `sticky`, only the **next** request gets that status, then the mock resets to 200.

```bash
# One-shot: next request returns 500, then reset
curl -s "http://127.0.0.1:19999/control?status=500"

# Next request sleeps 5s (timeout testing)
curl -s "http://127.0.0.1:19999/control?delay=5"

# Set eth_blockNumber result (decimal)
curl -s "http://127.0.0.1:19999/control?block=5000"
```

## Per-request headers (when calling mock directly)

| Header          | Values                  | Effect                                   |
|-----------------|-------------------------|------------------------------------------|
| `X-Mock-Status` | `200`, `400`, `429`, `500` | HTTP status to return (default `200`)   |
| `X-Mock-Delay`  | seconds, e.g. `5`       | Wait this many seconds before replying  |

## Examples

```bash
# 200 (success) - default
curl -X POST http://127.0.0.1:19999 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 500 - server error
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Status: 500" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 429 - rate limit
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Status: 429" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 10s delay - timeout testing
curl -X POST http://127.0.0.1:19999 \
  -H "X-Mock-Delay: 10" -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```
