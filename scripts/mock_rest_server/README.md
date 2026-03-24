# Mock REST Server

A simple HTTP server that returns configurable JSON responses for testing Cosmos REST endpoints.

**Key Feature:** The server automatically returns proper responses for validation endpoints (like `/cosmos/base/tendermint/v1beta1/blocks/latest`) so consumers can start up successfully, while returning configurable responses (default: node error) for other endpoints like `/cosmos/tx/v1beta1/txs`.

## Usage

### Start the server

```bash
# Start on default port 9999
go run ./scripts/mock_rest_server/main.go

# Or specify a custom port
go run ./scripts/mock_rest_server/main.go -port 8888
```

### Automatic validation responses

The following endpoints return proper mock responses automatically (for startup validation):
- `/cosmos/base/tendermint/v1beta1/blocks/latest` - Returns mock block
- `/cosmos/base/tendermint/v1beta1/blocks/{height}` - Returns mock block for any height
- `/cosmos/base/tendermint/v1beta1/node_info` - Returns mock node info
- `/cosmos/base/tendermint/v1beta1/syncing` - Returns `syncing: false`

All other endpoints (like `/cosmos/tx/v1beta1/txs`) return the configured response (default: node error).

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

## Example Scenarios

### Scenario 1: All providers fail with node errors

```bash
curl 'http://localhost:9999/control?type=sequence_error'
```

### Scenario 2: HTTP failures vs Node errors

```bash
# HTTP 500 (connection level error)
curl 'http://localhost:9999/control?status=500'

# vs HTTP 200 with node error (application level error)
curl 'http://localhost:9999/control?status=200&type=sequence_error'
```
