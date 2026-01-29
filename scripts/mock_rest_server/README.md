# Mock REST Server for Testing Smart Router

A simple HTTP server that returns configurable JSON responses for testing Cosmos REST endpoints.

## Usage

### Start the server

```bash
# Start on default port 9999
go run ./scripts/mock_rest_server/main.go

# Or specify a custom port
go run ./scripts/mock_rest_server/main.go -port 8888
```

### Configure responses via control endpoint

#### Pre-configured response types:

```bash
# Return sequence mismatch error (HTTP 200 with SDK error code 32)
curl 'http://localhost:9999/control?type=sequence_error'

# Return success response
curl 'http://localhost:9999/control?type=success'

# Return insufficient fee error
curl 'http://localhost:9999/control?type=insufficient_fee'
```

#### Custom HTTP status:

```bash
# Return HTTP 500
curl 'http://localhost:9999/control?status=500'

# Reset to HTTP 200
curl 'http://localhost:9999/control?status=200'
```

#### Custom response body:

```bash
# Set custom JSON response
curl 'http://localhost:9999/control?response=%7B%22custom%22%3A%22response%22%7D'
```

### Check current configuration

```bash
curl 'http://localhost:9999/control'
```

## Testing with Smart Router

### 1. Configure smart router to use mock server

Edit `smartrouter_lava.yml`:

```yaml
  - name: "lava-local-rest-3"
    chain-id: "LAV1"
    api-interface: "rest"
    node-urls:
      - url: "http://127.0.0.1:9999"
        skip-verifications:
          - chain-id
          - pruning
```

### 2. Restart smart router

Kill and restart your smart router process to pick up the new configuration.

### 3. Send a transaction

The mock server will return a node error (sequence mismatch) while other providers succeed:

```bash
# Send transaction through smart router
TX_BYTES=$(cat tx_bytes.txt)
curl -H "lava-relay-timeout: 10s" \
  -s -X POST "http://127.0.0.1:3360/cosmos/tx/v1beta1/txs" \
  -H "Content-Type: application/json" \
  -d "{\"tx_bytes\":\"$TX_BYTES\",\"mode\":\"BROADCAST_MODE_SYNC\"}" | jq
```

### 4. Check the logs

In `testutil/debugging/logs/SMARTROUTER_LAVA.log`, you should see:
- Provider 3 (mock) returns immediately with node error
- Providers 1 & 2 continue processing
- Smart router waits for success and returns it
- `nodeErrorCount=1` and `successCount=2` in quorum validation

## Example Scenarios

### Scenario 1: All providers fail with node errors

```bash
# Configure all providers to return errors
curl 'http://localhost:9999/control?type=sequence_error'
```

### Scenario 2: Mix of errors and success

Keep mock on error, real providers will succeed - tests that smart router prefers success over errors.

### Scenario 3: HTTP failures vs Node errors

```bash
# HTTP 500 (connection level error)
curl 'http://localhost:9999/control?status=500'

# vs HTTP 200 with node error (application level error)
curl 'http://localhost:9999/control?status=200&type=sequence_error'
```

The smart router should handle these differently - HTTP errors might trigger retries, while node errors should be detected as deterministic failures.
